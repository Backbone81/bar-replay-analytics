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
	go build ./cmd/bar-replay-analytics

.PHONY: test
test: prepare
	go test ./...

.PHONY: fuzz
fuzz: prepare
	PACKAGE='$(PACKAGE)' FUZZ_TARGET='$(FUZZ_TARGET)' FUZZ_IDLE_MINUTES='$(FUZZ_IDLE_MINUTES)' FUZZ_MAX_MINUTES='$(FUZZ_MAX_MINUTES)' FUZZ_PARALLEL='$(FUZZ_PARALLEL)' scripts/fuzz.sh

# GoReleaser takes the current and previous tag from git
release-test release-notes release: export GORELEASER_CURRENT_TAG = $(shell git describe --tags --abbrev=0)
release-test release-notes release: export GORELEASER_PREVIOUS_TAG = $(shell git describe --tags --abbrev=0 '$(GORELEASER_CURRENT_TAG)^')

.PHONY: release-test
release-test:
	goreleaser check
	goreleaser healthcheck
	goreleaser build --snapshot --clean
	goreleaser release --snapshot --clean --skip=publish

.PHONY: release-notes
release-notes:
	mkdir -p tmp
	scripts/release-notes.sh > tmp/release-notes.md
	@echo
	@echo "========== Release notes =========="
	@cat tmp/release-notes.md

.PHONY: release
release: release-test release-notes
release: export GITHUB_TOKEN ?= unknown
release:
	goreleaser release --clean --release-notes tmp/release-notes.md
