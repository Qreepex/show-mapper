# show-mapper

**Bridge physical control surfaces to live-event software.** Plug in a MIDI
board (Akai APC mini mk2 out of the box, any board via custom profiles -
Elgato Stream Deck next), press a button, and things happen on your show
network: grandMA3 commands/keys/faders via OSC today, ArtNet/sACN and
timecode tomorrow.

Single Go binary. Embedded web UI for configuration. No drivers to install.

**Core is completely generic.** Sources (MIDI boards, Stream Decks…), targets
(OSC, ArtNet…) and helper modules (e.g. grandMA3 action presets) are
self-contained modules under `internal/sources|targets|helpers/<name>` - each
with its *own* README. The binary runs fine with any subset of modules
compiled in; nothing device- or console-specific lives at the top level.

```
┌──────────────┐  USB   ┌───────────────────────┐  network  ┌───────────────┐
│ APC mini mk2 ├───────►│       show-mapper      ├──────────►│  grandMA3     │
│ custom board │        │  go backend + web UI  │   OSC     │  console/onPC │
│ Stream Deck* │        │  (ws realtime config) │           └───────────────┘
└──────────────┘        └───────────────────────┘  ArtNet/sACN*, MTC/LTC*
                                                     (* roadmap)
```

## Quick start (user)

1. Grab a release for your OS
   (`show-mapper_<ver>_windows_amd64.zip`, `_linux_amd64.tar.gz`,
   `_linux_arm64.tar.gz`, `_darwin_arm64.tar.gz`, `_darwin_amd64.tar.gz`) and
   extract it anywhere — no installer.
2. Run it (`show-mapper.exe` / `./show-mapper`). On first start a minimal
   `show-mapper.yaml` is generated automatically next to it.
3. Open the printed URL (default <http://127.0.0.1:8080>); everything else is
   done in the web UI. Hand edits of the YAML hot-reload without a restart.
4. **Mappings** page: wire button presses/holds/releases/faders to actions.
   The Dashboard shows a live event ticker - great for discovering controls.

> **Do I need drivers for my APC board?** No. The APC family (and almost all
> USB MIDI controllers) are *USB-MIDI class-compliant*: Windows (WinMM),
> macOS (CoreMIDI) and Linux (ALSA) expose them with in-box drivers.
> show-mapper talks to the OS MIDI stack via RtMidi. Each board model needs a
> **profile** (control map) in the software, not a driver - built-ins for
> APC mini/mk2 ship now; others are described per config or UI
> (docs/midi-devices.md).

## Quick start (developer)

```bash
# prerequisites: Go 1.24+, Node 22+; a C++ toolchain only for MIDI-enabled builds
git clone <repo> && cd show-tools

make web          # build the Svelte 5 UI (into internal/server/dist)
make build-nocgo  # fast path: binary without MIDI hardware support
make build        # full binary with MIDI (needs gcc/clang; see docs/releasing.md)

# dev loop with hot-reload UI:
make run          # terminal 1: backend API+WS on :8080 (stub MIDI w/o CGO)
make web-dev      # terminal 2: vite dev server on :5173 (proxy /api,/ws)

make check        # vet + unit tests + svelte-check + builds (same as CI)
```

Handy: `show-mapper midi list`, `show-mapper midi monitor <device>`.

## Feature status

| Area | Status |
| --- | --- |
| Sources | **MIDI** (APC mini mk2 verified protocol, APC mini community map, custom boards via config/UI) · Stream Deck (USB HID) planned next - see TODO.md |
| Triggers/Modes | pressed · released · hold (ms) · value · momentary · toggle (pad-LED feedback) |
| Targets | **OSC/UDP** (generic module; per-instance NIC bind via `localAddress` — multiple NICs in parallel) · grandMA3 helper module with action presets (Go/GoBack/Pause/Flash/Temp/On/Off/Key/Fader) |
| Realtime UI | WebSocket ticker + connector status; config hot-reload on UI save *and* disk edits |
| Config | single portable YAML (auto-detected next to the binary) · full **and per-section** UI import/export (bindings·sources·targets·boards) · hand-editable |
| Types | backend ⇄ frontend types generated from Go (`make types`, tygo) |
| Updates | optional self-update from GitHub releases (UI check + in-place install, checksum-verified) |
| Roadmap | ArtNet / sACN targets, MTC/LTC/ArtNet timecode, OSC feedback → LEDs, MIDI learn→profile wizard, hot-plug detection |

## Documentation

- **[docs/architecture.md](docs/architecture.md)** - the architecture & build plan, naming conventions, extension guides
- [docs/midi-devices.md](docs/midi-devices.md) - drivers FAQ, board profiles, adding boards
- Module docs live inside modules: [`internal/targets/osc`](internal/targets/osc/README.md) (generic OSC) · [`internal/helpers/gma3`](internal/helpers/gma3/README.md) (grandMA3 setup + presets)
- [docs/protocols.md](docs/protocols.md) - REST + WebSocket reference
- [docs/releasing.md](docs/releasing.md) - versioning, CI releases, CGO per OS
- [CONTRIBUTING.md](CONTRIBUTING.md) · [AGENTS.md](AGENTS.md) (agent/AI-assistant guide)

## Repo layout (short)

```
cmd/show-mapper/        CLI entry (serve, config init, midi list|monitor, version)
internal/core/         domain model: Source/Target interfaces, registry, conductor
internal/config/       YAML config model + validation
internal/sources/midi/ MIDI connector (RtMidi, CGO-gated) + board profiles
internal/targets/osc/  OSC/UDP target module
internal/helpers/gma3/ grandMA3 action-preset helper module
internal/server/         REST + WebSocket hub + embedded SPA
internal/updater/        self-update (GitHub releases, checksum-verified)
internal/version/        ldflags-stamped build info
web/                     Svelte 5 (runes) UI, SvelteKit SPA → embedded in backend
.github/workflows/     ci + release pipelines
```

*Name is tentative - only the module path needs a sweep if it changes.*

## License

MIT (see [LICENSE](LICENSE)).
