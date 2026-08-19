# Building & releasing show-mapper

## Why CGO (and what that means)

Real device I/O uses RtMidi (C++) → Go needs **CGO** (`CGO_ENABLED=1` plus a
C/C++ toolchain). Consequence: **we do not cross-compile releases** - CI
builds each platform natively on its own runner. For day-to-day dev and all
unit tests, `CGO_ENABLED=0` compiles the whole app with a stub MIDI driver
(status shows a clear error instead of hardware access), so frontend/API work
never needs a toolchain.

## Build prerequisites per OS

| Target | Toolchain notes |
| --- | --- |
| linux/amd64 | `apt install build-essential libasound2-dev pkg-config` (RtMidi uses ALSA) |
| windows/amd64 | MinGW g++ (`choco install mingw`; CI installs per job).**Release builds link `-extldflags=-static`** — no libstdc++/libgcc/winpthread DLLs needed on end-user machines. |
| darwin/amd64, darwin/arm64 | Xcode CLT (present on GH runners); links CoreMIDI/CoreAudio frameworks |

Local builds: `make build` (CGO) / `make build-nocgo` (portable). Version info
goes in via ldflags (`internal/version`): `-X …/version.Version=0.1.0
-X …/version.Commit=<sha> -X …/version.Date=<rfc3339>`.

## Versioning & releases

- **Semver** via git tags `vX.Y.Z` (pre-1.0: minor = features, patch = fixes).
- Tag on `main` → push → `.github/workflows/release.yml` runs the matrix
  (linux/amd64, linux/arm64, windows/amd64, darwin/amd64, darwin/arm64):
  per OS: install toolchain → build web (`npm ci && npm run build`) →
  `CGO_ENABLED=1 go build -trimpath -ldflags …` → package
  `show-mapper_<ver>_<os>_<arch>.{zip,tar.gz}` (+ README/LICENSE/example
  config) → `checksums.txt` → GitHub Release (auto notes).
- Practically:

  ```bash
  git tag v0.1.0 && git push origin v0.1.0   # CI does the rest
  ```

- Local snapshot (single platform): `make web && make build` - or run the
  release workflow's build step manually.

## Adding a platform

Edit `release.yml`'s matrix. Watch-outs: every entry builds **natively** with a
matching toolchain. linux/arm64 uses the hosted ARM runner
(`ubuntu-24.04-arm`). windows/arm64 isn't wired up yet: GH hosted
`windows-11-arm` runners work, but RtMidi needs a working clang-mingw
toolchain there - add when there's real demand (❗ verify `midi list` on the
resulting binary).

## Code signing / quarantine (current: unsigned)

- Missing-DLL error (`libstdc++-6.dll not found`)? That means a *dynamically linked* local build. Release binaries are self-contained (static C/C++ runtime). For local dev builds either copy MinGW's DLLs next to the exe or pass `-ldflags '-extldflags=-static'`.

- macOS: binaries are unsigned → first run: `xattr -d com.apple.quarantine`
  or right-click → Open. Notarization is a future (paid-cert) topic.
- Windows: unsigned → SmartScreen warning; same story (OV cert later).
- Linux: fine as-is; distro packages are out of scope for now.

## Self-update channel

The app can update itself from these very release assets
(`internal/updater`, `rhysd/go-github-selfupdate`): config `updates.repo`
pointing at the GitHub repo + optional `autoCheck` on startup. The updater
verifies downloads against the workflow-generated `checksums.txt`.
On Windows the replaced binary is moved aside (`show-mapper.old.exe`);
delete it once the new version runs. Keep asset naming
(`show-mapper_<ver>_<os>_<arch>.{zip,tar.gz}`) stable - the updater's asset
detection depends on it.

## CI (non-release)

`.github/workflows/ci.yml` on push/PR: gofmt-checked via golangci, `go vet`,
`go test` (CGO off), `go build` (CGO off, embeds placeholder), `npm ci`,
`npm run check` (svelte-check = 0 findings policy), `npm run build` +
final `go build` with real SPA, **generated-types freshness**
(`go tool tygo generate && git diff --exit-code`). Dependabot watches go.mod,
web/npm and actions weekly (`.github/dependabot.yml`).

## Branching

`main` = always releasable; feature branches → PR (template provided) →
squash merge. Tags only on main.
