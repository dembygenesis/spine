package testsuite

import (
	"log"
	"os"
	"strconv"
	"sync"

	"github.com/dembygenesis/spine/pkg/pg"
)

// A pool of Postgres containers, sized by how hard this package leans on the
// database.
//
// One server has one ceiling: max_connections, and one pg_database catalog that
// every CREATE DATABASE and DROP DATABASE queues on. Separate servers have
// separate ceilings and separate catalogs.
//
// go test runs each package as its own process, so this state is per package:
// each counts its own requests and grows on its own. Two packages wanting three
// containers do not end up with six, because the names are global and the
// second attaches to what the first started.
//
// Growth is deterministic: the same tests make the same requests and reach the
// same size every run, which keeps a failure reproducible.
type pool struct {
	mu        sync.Mutex
	asks      int
	size      int // containers in use; grows, never shrinks
	cursor    int // round-robin position
	max       int
	growEvery int
}

const (
	// EnvTestContainers caps the pool.
	EnvTestContainers = "SPINE_TEST_CONTAINERS"
	// EnvTestGrowEvery is how many database requests earn another container.
	EnvTestGrowEvery = "SPINE_TEST_GROW_EVERY"

	// A new container costs a boot and a template build, a few seconds, so
	// growth is kept rare: only a package making 80 requests grows at all.
	defaultMaxContainers = 3
	defaultGrowEvery     = 80
	hardMaxContainers    = 10
)

// dbPool is this package's pool. One process, one pool.
var dbPool = newPool()

func newPool() *pool {
	return &pool{
		size:      1,
		max:       envInt(EnvTestContainers, defaultMaxContainers, 1, hardMaxContainers),
		growEvery: envInt(EnvTestGrowEvery, defaultGrowEvery, 1, 1000),
	}
}

// next returns the container this request should use.
func (p *pool) next() (*pg.TestContainer, error) {
	return pg.GetContainerN(p.reserve())
}

// reserve counts the request, grows the pool if the request earned it, and
// returns the container index to use. It is split from next so the growth
// rule, the part most likely to be wrong, is testable without Docker.
func (p *pool) reserve() int {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.asks++
	if p.size < p.max && p.asks >= p.size*p.growEvery {
		p.size++
		log.Printf("[testsuite] pool grew to %d container(s) after %d requests", p.size, p.asks)
	}
	idx := p.cursor % p.size
	p.cursor++
	return idx
}

// envInt reads a tunable and clamps it. A typo in an environment variable picks
// the nearest sane value and says so, rather than refusing to run the suite.
func envInt(key string, def, lo, hi int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return def
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		log.Printf("[testsuite] %s=%q is not a number, using %d", key, raw, def)
		return def
	}
	if n < lo {
		log.Printf("[testsuite] %s=%d is below %d, using %d", key, n, lo, lo)
		return lo
	}
	if n > hi {
		log.Printf("[testsuite] %s=%d is above %d, using %d", key, n, hi, hi)
		return hi
	}
	return n
}
