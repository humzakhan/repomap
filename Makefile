BINARY := repomap
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X github.com/repomap/repomap/cmd.version=$(VERSION)"

.PHONY: build test lint clean all test-integration

build:
	go build $(LDFLAGS) -o $(BINARY) .

test:
	go test ./... -count=1

test-integration:
	REPOMAP_INTEGRATION=1 go test -tags integration ./... -count=1

lint:
	golangci-lint run ./...

clean:
	rm -f $(BINARY)

all: build
