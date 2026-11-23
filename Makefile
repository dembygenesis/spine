.PHONY: test check fmt-check vet grade check-commits

# Every test that touches Postgres needs Docker running.
test:
	go test ./... -count=1

vet:
	go vet ./...

fmt-check:
	@out=$$(gofmt -l .); if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi

# What every commit must pass.
check: fmt-check vet test

# make grade LESSON=03 runs lesson 03's tests against your working tree.
grade:
	@scripts/grade.sh $(LESSON)

# make check-commits lints your commit messages since lesson-00.
check-commits:
	@scripts/check-commits.sh
