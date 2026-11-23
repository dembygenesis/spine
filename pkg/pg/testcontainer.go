// Package pg holds the Postgres plumbing shared by the app and its tests.
package pg

import (
	"context"
	"fmt"
	"os"
	"strconv"
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

// ContainerName is the fixed name of the first shared test container. A fixed
// name is what lets every test binary in a run attach to one container instead
// of starting its own.
const ContainerName = "spine-test-pg"

// containerName is the name of container n: spine-test-pg, spine-test-pg-1, ...
func containerName(n int) string {
	if n == 0 {
		return ContainerName
	}
	return ContainerName + "-" + strconv.Itoa(n)
}

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

// One sync.Once per container index: container 2 may be wanted long after
// container 0 started, and each must start exactly once however many parallel
// tests ask for it at the same moment.
var (
	slotsMu sync.Mutex
	slots   = map[int]*slot{}
)

type slot struct {
	once      sync.Once
	container *TestContainer
	err       error
}

func slotFor(n int) *slot {
	slotsMu.Lock()
	defer slotsMu.Unlock()
	if s, ok := slots[n]; ok {
		return s
	}
	s := &slot{}
	slots[n] = s
	return s
}

// GetContainer returns the first shared test container, starting it on first
// use.
func GetContainer() (*TestContainer, error) {
	return GetContainerN(0)
}

// GetContainerN returns container n, starting it on first use.
//
// sync.Once makes parallel tests in one binary share a single start. Reuse makes
// other binaries in the same run (go test runs one per package) attach to the
// container by name rather than racing to start their own, so n counts
// containers for the whole run, not per package.
func GetContainerN(n int) (*TestContainer, error) {
	s := slotFor(n)
	s.once.Do(func() {
		s.container, s.err = startContainer(n)
	})
	return s.container, s.err
}

func startContainer(n int) (*TestContainer, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	name := containerName(n)
	req := testcontainers.ContainerRequest{
		Name:         name,
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
		// The data directory lives in RAM: CREATE DATABASE ... TEMPLATE is a
		// file copy, and disk latency is what it waits on.
		Tmpfs: map[string]string{"/var/lib/postgresql/data": "rw,size=2g"},
		// Every setting below is reckless in production and free here: a test
		// database lives for a second and is dropped, so a crash loses nothing.
		Cmd: []string{
			"postgres",
			"-c", "fsync=off", // never flush to disk
			"-c", "synchronous_commit=off", // do not wait on the WAL either
			"-c", "full_page_writes=off", // no torn-page protection needed
			"-c", "max_connections=300", // each running test holds two
		},
	}

	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
		Reuse:            true,
	})
	if err != nil {
		return nil, fmt.Errorf("start %s: %w", name, err)
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
