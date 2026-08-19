# showbridge — developer quick commands. Works in bash (Linux/macOS/git-bash).
# Requires: Go 1.24+, Node 22+ (web/), and for MIDI-enabled builds a C++
# toolchain (see docs/releasing.md).

MODULE   := github.com/yourorg/showbridge
BIN      := bin/showbridge
ifeq ($(OS),Windows_NT)
BIN := bin/showbridge.exe
endif

VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null | sed s/^v// || echo dev)
COMMIT   ?= $(shell git rev-parse HEAD 2>/dev/null || echo none)
DATE     ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS  := -s -w -X $(MODULE)/internal/version.Version=$(VERSION) \
                  -X $(MODULE)/internal/version.Commit=$(COMMIT) \
                  -X $(MODULE)/internal/version.Date=$(DATE)

.PHONY: all build build-nocgo run web web-webdev dev test vet lint check clean midi-list

## all: frontend + backend binary (MIDI enabled if a C++ toolchain is present)
all: web build

## build: backend with CGO (real MIDI support) — needs gcc/clang toolchain
build:
	mkdir -p bin
	CGO_ENABLED=1 go build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/showbridge
	@echo "built $(BIN) ($(VERSION))"

## build-nocgo: backend without MIDI hardware support (compiles anywhere)
build-nocgo:
	mkdir -p bin
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/showbridge
	@echo "built $(BIN) ($(VERSION), no MIDI)"

## run: run backend (serves API+UI on :8080); placeholder UI until `make web`
run:
	CGO_ENABLED=0 go run ./cmd/showbridge serve

## web: build SPA into internal/server/dist (embedded by go build)
web:
	cd web && npm ci && npm run build

## web-dev: vite dev server on :5173 with /api+/ws proxy → backend on :8080
web-dev:
	cd web && npm ci && npm run dev

## dev: print the two-terminal dev loop
dev:
	@echo "terminal 1:  make run        (backend :8080)"
	@echo "terminal 2:  make web-dev    (UI :5173, hot reload)"

test:
	CGO_ENABLED=0 go test ./...

vet:
	go vet ./...

lint:
	golangci-lint run ./...

## check: everything CI runs locally (except the OS matrix)
check: vet test
	cd web && npm run check && npm run build
	CGO_ENABLED=0 go build ./...

midi-list:
	CGO_ENABLED=0 go run ./cmd/showbridge midi list || true

clean:
	rm -rf bin web/.svelte-kit internal/server/dist/_app internal/server/dist/favicon.svg \
		internal/server/dist/*.html
	git checkout -- internal/server/dist/ 2>/dev/null || true
