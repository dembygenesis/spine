// Package config reads the process environment once, at startup, and refuses
// to start on anything missing.
package config

import (
	"fmt"
	"strconv"
	"strings"
)

// Environment variables the binary reads.
const (
	EnvDBHost    = "SPINE_DB_HOST"
	EnvDBPort    = "SPINE_DB_PORT"
	EnvDBUser    = "SPINE_DB_USER"
	EnvDBPass    = "SPINE_DB_PASS"
	EnvDBName    = "SPINE_DB_NAME"
	EnvJWTSecret = "SPINE_JWT_SECRET"
	EnvPort      = "SPINE_PORT"
)

// Config is everything the app needs to start.
type Config struct {
	DBHost    string
	DBPort    int
	DBUser    string
	DBPass    string
	DBName    string
	JWTSecret string
	Port      int
}

// DSN is the database connection string.
func (c *Config) DSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		c.DBHost, c.DBPort, c.DBUser, c.DBPass, c.DBName)
}

// Load reads the config through getenv (os.Getenv in production, a map in
// tests). Every missing variable is reported at once: fixing one per restart is
// how a deploy takes an afternoon.
func Load(getenv func(string) string) (*Config, error) {
	var missing []string
	need := func(key string) string {
		v := getenv(key)
		if v == "" {
			missing = append(missing, key)
		}
		return v
	}

	c := &Config{
		DBHost:    need(EnvDBHost),
		DBUser:    need(EnvDBUser),
		DBPass:    need(EnvDBPass),
		DBName:    need(EnvDBName),
		JWTSecret: need(EnvJWTSecret),
	}
	dbPort, port := need(EnvDBPort), need(EnvPort)
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing environment: %s", strings.Join(missing, ", "))
	}

	var err error
	if c.DBPort, err = strconv.Atoi(dbPort); err != nil {
		return nil, fmt.Errorf("%s=%q is not a port", EnvDBPort, dbPort)
	}
	if c.Port, err = strconv.Atoi(port); err != nil {
		return nil, fmt.Errorf("%s=%q is not a port", EnvPort, port)
	}
	return c, nil
}
