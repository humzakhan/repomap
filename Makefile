BINARY := repomap
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X github.com/repomap/repomap/cmd.version=$(VERSION)"

.PHONY: build test lint clean all test-integration frontend

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
	rm -rf internal/renderer/dist

frontend:
	cd report && npm install && npm run build
	mkdir -p internal/renderer/dist
	cp report/dist/bundle.js internal/renderer/dist/bundle.js
	cp report/dist/styles.css internal/renderer/dist/styles.css

all: frontend build
