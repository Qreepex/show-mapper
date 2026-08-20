# show-mapper - developer quick commands. Works in bash (Linux/macOS/git-bash).
# Requires: Go 1.24+, Node 22+ (web/), and for MIDI-enabled builds a C++
# toolchain (see docs/releasing.md).

MODULE   := github.com/Qreepex/show-mapper
BIN      := bin/show-mapper
ifeq ($(OS),Windows_NT)
BIN := bin/show-mapper.exe
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

## build: backend with CGO (real MIDI support) - needs gcc/clang toolchain
build:
	mkdir -p bin
	CGO_ENABLED=1 go build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/show-mapper
	@echo "built $(BIN) ($(VERSION))"

## build-nocgo: backend without MIDI hardware support (compiles anywhere)
build-nocgo:
	mkdir -p bin
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/show-mapper
	@echo "built $(BIN) ($(VERSION), no MIDI)"

## build-windows: cross-build the Windows exe from Linux/WSL (mingw-w64) or
## build natively on Windows with MinGW (both need a C toolchain for MIDI).
## Static runtime link → exe runs without MinGW DLLs.
build-windows:
	mkdir -p bin
	@if command -v x86_64-w64-mingw32-g++ >/dev/null 2>&1; then \
		echo "cross-compiling Windows exe via mingw-w64"; \
		CGO_ENABLED=1 GOOS=windows GOARCH=amd64 \
		CC=x86_64-w64-mingw32-gcc CXX=x86_64-w64-mingw32-g++ \
		go build -trimpath -ldflags "-s -w -extldflags=-static $(LDFLAGS)" -o bin/show-mapper.exe ./cmd/show-mapper; \
	elif command -v g++ >/dev/null 2>&1; then \
		echo "native MinGW build"; \
		CGO_ENABLED=1 go build -trimpath -ldflags "-s -w -extldflags=-static $(LDFLAGS)" -o bin/show-mapper.exe ./cmd/show-mapper; \
	else \
		echo "need a C++ toolchain: apt install g++-mingw-w64 (WSL/Linux) or choco install mingw (Windows)"; exit 1; \
	fi
	@echo "built bin/show-mapper.exe"

## run: run backend (serves API+UI on :8484). Uses CGO (real MIDI) when the
## full native toolchain is available (Linux additionally needs ALSA headers),
## otherwise falls back to the stub + virtual Surface for development.
run:
	@if (command -v gcc >/dev/null 2>&1 || command -v cc >/dev/null 2>&1 || command -v clang >/dev/null 2>&1) && \
	    { [ "$$(uname -s)" != "Linux" ] || { command -v pkg-config >/dev/null 2>&1 && pkg-config --exists alsa; }; }; then \
		echo "toolchain OK → MIDI enabled (CGO_ENABLED=1)"; \
		CGO_ENABLED=1 go run ./cmd/show-mapper serve; \
	else \
		echo "no full MIDI toolchain → building WITHOUT MIDI hardware support"; \
		if [ "$$(uname -s)" = "Linux" ]; then \
			echo "  (Linux: sudo apt-get install -y build-essential libasound2-dev pkg-config)"; \
		fi; \
		echo "  → use the built-in virtual surface meanwhile: http://127.0.0.1:8484/surface"; \
		CGO_ENABLED=0 go run ./cmd/show-mapper serve; \
	fi

## web: build SPA into internal/server/dist (embedded by go build)
web:
	cd web && npm ci && npm run build

## web-dev: vite dev server on :5173 with /api+/ws proxy → backend on :8484
web-dev:
	cd web && npm ci && npm run dev

## dev: print the two-terminal dev loop
dev:
	@echo "terminal 1:  make run        (backend :8484)"
	@echo "terminal 2:  make web-dev    (UI :5173, hot reload)"

test:
	CGO_ENABLED=0 go test ./...

## types: regenerate backend→frontend wire types (tygo) into web/src/lib/generated
types:
	go tool tygo generate

vet:
	go vet ./...

lint:
	golangci-lint run ./...

## check: everything CI runs locally (except the OS matrix)
check: vet test
	go tool tygo generate
	git diff --exit-code -- web/src/lib/generated
	cd web && npm run check && npm run build
	CGO_ENABLED=0 go build ./...

midi-list:
	CGO_ENABLED=0 go run ./cmd/show-mapper midi list || true

clean:
	rm -rf bin web/.svelte-kit internal/server/dist/_app internal/server/dist/favicon.svg \
		internal/server/dist/*.html
	git checkout -- internal/server/dist/ 2>/dev/null || true
