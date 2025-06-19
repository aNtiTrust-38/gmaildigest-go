
# Makefile for Gmail Digest Assistant v3.0

VERSION ?= $(shell git describe --tags --always --dirty)
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.buildTime=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)

.PHONY: test build build-prod build-static cross-compile docker clean lint security-scan install uninstall check-pr-labels

# Development build
build:
	go build -ldflags="$(LDFLAGS)" -o bin/gda cmd/gda/main.go

# Production build with optimizations
build-prod:
	CGO_ENABLED=1 go build -ldflags="$(LDFLAGS)" -trimpath -o bin/gda cmd/gda/main.go

# Static binary for containerless deployment
build-static:
	CGO_ENABLED=1 go build -ldflags="$(LDFLAGS) -extldflags=-static" -trimpath -o bin/gda-static cmd/gda/main.go

# Cross-compilation targets
cross-compile:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -ldflags="$(LDFLAGS)" -o bin/gda-linux-amd64 cmd/gda/main.go
	GOOS=linux GOARCH=arm64 CGO_ENABLED=1 go build -ldflags="$(LDFLAGS)" -o bin/gda-linux-arm64 cmd/gda/main.go
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 go build -ldflags="$(LDFLAGS)" -o bin/gda-darwin-amd64 cmd/gda/main.go
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=1 go build -ldflags="$(LDFLAGS)" -o bin/gda-darwin-arm64 cmd/gda/main.go

# Testing targets
test:
	go test -v -race -coverprofile=coverage.out ./...

test-integration:
	go test -v -tags=integration ./test/integration/...
    
benchmark:
	go test -bench=. -benchmem ./...

# Linting and quality checks
lint:
	golangci-lint run ./...

security-scan:
	gosec ./...

# Deployment helpers
install: build-prod
	sudo ./scripts/install.sh

uninstall:
	sudo ./scripts/uninstall.sh

clean:
	rm -rf bin/ coverage.out

# PR Label Check
check-pr-labels:
	@gh pr view --json labels --jq '.labels[].name' | grep -Eq 'M[1-9]|Final' || \
	(echo "❌ PR must have at least one milestone label (M1..M9 or Final)" && exit 1)
