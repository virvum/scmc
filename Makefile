BINARY_NAME = scmc
LDFLAGS = -s -w \
          -X 'main.rootPath=$$(pwd)' \
          -X 'main.buildVersion=$$(git describe 2>/dev/null || git rev-parse --short HEAD) ($$(git rev-parse --abbrev-ref HEAD))' \
          -X 'main.buildDate=$$(date -u '+%Y-%m-%d %H:%M:%S %Z')'

help:
	@echo '$(BINARY_NAME) Makefile'
	@echo
	@echo 'Usage:'
	@echo
	@echo '  make build    Build production binary.'
	@echo '  make install  Build and install production binary.'
	@echo '  make godoc    Run local `godoc` HTTP service.'
	@echo '  make gofmt    Run `go fmt` on all Go files.'
	@echo '  make govet    Run `go vet` on all Go files.'
	@echo

build:
	ENABLE_CGO=0 go build -ldflags "$(LDFLAGS)" -o "$(BINARY_NAME)" cmd/scmc/*.go

install:
	cd cmd/scmc; go install -ldflags "$(LDFLAGS)"

godoc:
	godoc -http=:6060

gofmt:
	find . -type f -name '*.go' | xargs gofmt -s -e -d -w

golint:
	golint ./...

govet:
	find . -type f -name '*.go' | xargs dirname | sort -u | xargs go vet

.PHONY: help build install godoc gofmt golint govet
