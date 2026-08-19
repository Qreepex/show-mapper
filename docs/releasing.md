# Building & releasing showbridge

## Why CGO (and what that means)

Real device I/O uses RtMidi (C++) → Go needs **CGO** (`CGO_ENABLED=1` plus a
C/C++ toolchain). Consequence: **we do not cross-compile releases** — CI
builds each platform natively on its own runner. For day-to-day dev and all
unit tests, `CGO_ENABLED=0` compiles the whole app with a stub MIDI driver
(status shows a clear error instead of hardware access), so frontend/API work
never needs a toolchain.

## Build prerequisites per OS

| Target | Toolchain notes |
|---|---|
| linux/amd64 | `apt install build-essential libasound2-dev pkg-config` (RtMidi uses ALSA) |
| windows/amd64 | MinGW-w64 g++ (e.g. `choco install mingw` or the msys2 UCRT64 toolchain we install in CI) |
| darwin/amd64, darwin/arm64 | Xcode CLT (present on GH runners); links CoreMIDI/CoreAudio frameworks |

Local builds: `make build` (CGO) / `make build-nocgo` (portable). Version info
goes in via ldflags (`internal/version`): `-X …/version.Version=0.1.0
-X …/version.Commit=<sha> -X …/version.Date=<rfc3339>`.

## Versioning & releases

- **Semver** via git tags `vX.Y.Z` (pre-1.0: minor = features, patch = fixes).
- Tag on `main` → push → `.github/workflows/release.yml` runs the matrix:
  per OS: install toolchain → build web (`npm ci && npm run build`) →
  `CGO_ENABLED=1 go build -trimpath -ldflags …` → package
  `showbridge_<ver>_<os>_<arch>.{zip,tar.gz}` (+ README/LICENSE/example
  config) → `checksums.txt` → GitHub Release (auto notes).
- Practically:
  ```bash
  git tag v0.1.0 && git push origin v0.1.0   # CI does the rest
  ```
- Local snapshot (single platform): `make web && make build` — or run the
  release workflow's build step manually.

## Adding a platform

Edit `release.yml` matrix. Watch-outs: every entry builds **natively** with a
matching toolchain — e.g. linux/arm64 needs an ARM runner (or cgo
cross-toolchain + alsa sysroot: deliberately out of scope until requested);
windows/arm64 needs clang-mingw — same story.

## Code signing / quarantine (current: unsigned)

- macOS: binaries are unsigned → first run: `xattr -d com.apple.quarantine`
  or right-click → Open. Notarization is a future (paid-cert) topic.
- Windows: unsigned → SmartScreen warning; same story (OV cert later).
- Linux: fine as-is; distro packages are out of scope for now.

## CI (non-release)

`.github/workflows/ci.yml` on push/PR: gofmt-checked via golangci, `go vet`,
`go test` (CGO off), `go build` (CGO off, embeds placeholder), `npm ci`,
`npm run check` (svelte-check = 0 findings policy), `npm run build` +
final `go build` with real SPA. Dependabot watches go.mod, web/npm and
actions weekly (`.github/dependabot.yml`).

## Branching

`main` = always releasable; feature branches → PR (template provided) →
squash merge. Tags only on main.
