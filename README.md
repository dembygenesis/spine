# The Spine

A Go course that builds the core of a multi-tenant backend (organizations, users and the
test harness under them) the way production code is built: small commits, every one green.

Each lesson ends on a tag. `lesson-03` is the code exactly as lesson 3 leaves it.

## Follow along

```sh
git clone <this repo> spine && cd spine
git switch -c my-work lesson-02       # start lesson 3 from where lesson 2 ended
# ...build lesson 3...
make grade LESSON=03                  # lesson 3's tests, run against your code
make check-commits                    # your commit messages, linted
```

## Needs

- Go (see `go.mod`)
- Docker (OrbStack or Docker Desktop) from lesson 1 onward

## Reset the test containers

An interrupted run can leave containers behind. They are named `spine-test-pg*`:

```sh
docker ps -a --filter name=spine-test-pg
```
