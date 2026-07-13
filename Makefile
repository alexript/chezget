# Makefile for chezget
#
# Targets:
#   build        - build the chezget binary for the current host
#   test         - run the unit tests
#   cover        - run tests with coverage and print a summary
#   cover-html   - generate HTML coverage report in coverage.html
#   vet          - run go vet
#   fmt          - format the source tree
#   clean        - remove build artifacts
#   cross        - cross-compile binaries for all supported platforms
#   install      - install chezget into $GOPATH/bin

BINARY  := chezget
MODULE  := github.com/alexript/chezget
MAIN    := $(MODULE)/cmd/chezget
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X main.version=$(VERSION)

GOOS_LIST   := linux darwin windows freebsd
GOARCH_LIST := amd64 arm64

.PHONY: build test cover cover-html vet fmt clean cross install all
all: build

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/chezget

test:
	go test ./...

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -n 1

cover-html: cover
	go tool cover -html=coverage.out -o coverage.html

vet:
	go vet ./...

fmt:
	gofmt -s -w .

clean:
	rm -f $(BINARY) coverage.out coverage.html
	rm -rf dist

cross:
	@mkdir -p dist
	@for os in $(GOOS_LIST); do \
	  for arch in $(GOARCH_LIST); do \
	    ext=""; \
	    if [ $$os = windows ]; then ext=".exe"; fi; \
	    echo "  -> $$os/$$arch"; \
	    GOOS=$$os GOARCH=$$arch go build -ldflags "$(LDFLAGS)" \
	      -o dist/$(BINARY)-$$os-$$arch$$ext ./cmd/chezget || exit 1; \
	  done; \
	done

install:
	go install -ldflags "$(LDFLAGS)" ./cmd/chezget