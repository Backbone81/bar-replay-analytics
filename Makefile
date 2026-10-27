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
