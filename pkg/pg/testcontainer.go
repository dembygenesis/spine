// Package pg holds the Postgres plumbing shared by the app and its tests.
package pg

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func init() {
	// Ryuk is testcontainers' reaper: it removes a process's containers when
	// that process exits. go test runs one process per package, all sharing
	// one container by name, so the first package to finish would remove the
	// container under the others. The container outlives the run instead.
	if os.Getenv("TESTCONTAINERS_RYUK_DISABLED") == "" {
		_ = os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")
	}

	// testcontainers looks for Docker at /var/run/docker.sock. OrbStack and
	// newer Docker Desktop put the socket under $HOME instead.
	if os.Getenv("DOCKER_HOST") != "" {
		return
	}
	for _, sock := range []string{
		os.ExpandEnv("$HOME/.orbstack/run/docker.sock"),
		os.ExpandEnv("$HOME/.docker/run/docker.sock"),
	} {
		if _, err := os.Stat(sock); err == nil {
			_ = os.Setenv("DOCKER_HOST", "unix://"+sock)
			return
		}
	}
}

// ContainerName is the fixed name of the shared test container. A fixed name is
// what lets every test binary in a run attach to one container instead of
// starting its own.
const ContainerName = "spine-test-pg"

// ContainerConfig is how to reach the test container.
type ContainerConfig struct {
	Host     string
	Port     int
	User     string
	Pass     string
	Database string
}

// DSN is a connection string for dbName on this container.
func (c *ContainerConfig) DSN(dbName string) string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		c.Host, c.Port, c.User, c.Pass, dbName)
}

// TestContainer is a running Postgres the tests can connect to.
type TestContainer struct {
	Container testcontainers.Container
	Config    *ContainerConfig
}

var (
	containerOnce sync.Once
	container     *TestContainer
	containerErr  error
)

// GetContainer returns the shared test container, starting it on first use.
//
// sync.Once makes parallel tests in one binary share a single start. Reuse makes
// other binaries in the same run (go test runs one per package) attach to the
// container by name rather than racing to start their own.
func GetContainer() (*TestContainer, error) {
	containerOnce.Do(func() {
		container, containerErr = startContainer()
	})
	return container, containerErr
}

func startContainer() (*TestContainer, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	req := testcontainers.ContainerRequest{
		Name:         ContainerName,
		Image:        "postgres:15-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_DB":       "testdb",
		},
		// The official image starts Postgres twice: once to run its init
		// scripts, then for real. The second "ready" line is the real one.
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).
			WithStartupTimeout(time.Minute),
	}

	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
		Reuse:            true,
	})
	if err != nil {
		return nil, fmt.Errorf("start %s: %w", ContainerName, err)
	}

	host, err := c.Host(ctx)
	if err != nil {
		return nil, fmt.Errorf("container host: %w", err)
	}
	port, err := c.MappedPort(ctx, "5432")
	if err != nil {
		return nil, fmt.Errorf("container port: %w", err)
	}

	return &TestContainer{
		Container: c,
		Config: &ContainerConfig{
			Host:     host,
			Port:     port.Int(),
			User:     "test",
			Pass:     "test",
			Database: "testdb",
		},
	}, nil
}
