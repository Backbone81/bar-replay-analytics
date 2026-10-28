PACKAGE ?= ./internal/...

# The fuzz target runs scripts/fuzz.sh, whose header explains these variables. FUZZ_TARGET restricts the fuzz tests by
# name, FUZZ_IDLE_MINUTES is how long the corpus of a fuzz test may stop growing before it is done, FUZZ_MAX_MINUTES
# optionally caps every fuzz test and FUZZ_PARALLEL sets the number of workers, which is every core by default.
FUZZ_TARGET ?=
FUZZ_IDLE_MINUTES ?= 10
FUZZ_MAX_MINUTES ?=
FUZZ_PARALLEL ?=

export CGO_ENABLED=0

.PHONY: all
all: build

.PHONY: prepare
prepare:
	go mod tidy
	go fmt ./...
	go vet ./...
	docker run --tty --rm \
		--workdir /app \
		--user $$(id --user):$$(id --group) \
		--volume $$(pwd):/app \
		--volume $$(go env GOCACHE):/.cache/go-build \
		--volume $$(go env GOMODCACHE):/.cache/mod \
		--volume ~/.cache/golangci-lint:/.cache/golangci-lint \
		--env GOCACHE=/.cache/go-build \
		--env GOMODCACHE=/.cache/mod \
		--env GOLANGCI_LINT_CACHE=/.cache/golangci-lint \
		golangci/golangci-lint:v2.15.0 golangci-lint run --fix

.PHONY: build
build: prepare
	go build ./cmd/bar-replay-analytics-go

.PHONY: test
test: prepare
	go test ./...

.PHONY: fuzz
fuzz: prepare
	PACKAGE='$(PACKAGE)' FUZZ_TARGET='$(FUZZ_TARGET)' FUZZ_IDLE_MINUTES='$(FUZZ_IDLE_MINUTES)' FUZZ_MAX_MINUTES='$(FUZZ_MAX_MINUTES)' FUZZ_PARALLEL='$(FUZZ_PARALLEL)' scripts/fuzz.sh
