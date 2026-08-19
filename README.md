# showbridge

**Bridge physical control surfaces to live-event software.** Plug in a MIDI
board (Akai APC mini mk2 out of the box, any board via custom profiles —
Elgato Stream Deck next), press a button, and things happen on your show
network: grandMA3 commands/keys/faders via OSC today, ArtNet/sACN and
timecode tomorrow.

Single Go binary. Embedded web UI for configuration. No drivers to install.

```
┌──────────────┐  USB   ┌───────────────────────┐  network  ┌───────────────┐
│ APC mini mk2 ├───────►│       showbridge      ├──────────►│  grandMA3     │
│ custom board │        │  go backend + web UI  │   OSC     │  console/onPC │
│ Stream Deck* │        │  (ws realtime config) │           └───────────────┘
└──────────────┘        └───────────────────────┘  ArtNet/sACN*, MTC/LTC*
                                                     (* roadmap)
```

## Quick start (user)

1. Grab a release for your OS
   (`showbridge_<ver>_windows_amd64.zip`, `_linux_amd64.tar.gz`,
   `_darwin_arm64.tar.gz`, `_darwin_amd64.tar.gz`).
2. Run `showbridge config init`, edit `showbridge.yaml` (set your console IP).
3. Run `showbridge.exe` / `./showbridge` → open the printed URL
   (default <http://127.0.0.1:8080>).
4. **Mappings** page: wire button presses/holds/releases/faders to actions.
   The Dashboard shows a live event ticker — great for discovering controls.

> **Do I need drivers for my APC board?** No. The APC family (and almost all
> USB MIDI controllers) are *USB-MIDI class-compliant*: Windows (WinMM),
> macOS (CoreMIDI) and Linux (ALSA) expose them with in-box drivers.
> showbridge talks to the OS MIDI stack via RtMidi. Each board model needs a
> **profile** (control map) in the software, not a driver — built-ins for
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

Handy: `showbridge midi list`, `showbridge midi monitor <device>`.

## Feature status

| Area | Status |
|---|---|
| Sources | **MIDI** (APC mini mk2 verified protocol, APC mini community map, custom boards via config/UI) · Stream Deck (USB HID) planned next |
| Triggers | pressed · released · hold (ms) · value (fader/encoder) |
| Modes | momentary · toggle (with pad-LED feedback on supporting boards) |
| Targets | **OSC/UDP** (grandMA3 `/cmd`, `/Page1/FaderNNN`, `/Page1/KeyNNN` + prefix) |
| Realtime UI | WebSocket ticker + connector status, config hot-reload |
| Roadmap | ArtNet / sACN targets, MTC/LTC/ArtNet timecode, OSC feedback → LEDs, MIDI learn→profile wizard, hot-plug detection |

## Documentation

- **[docs/architecture.md](docs/architecture.md)** — the architecture & build plan, naming conventions, extension guides
- [docs/midi-devices.md](docs/midi-devices.md) — drivers FAQ, board profiles, adding boards
- [docs/grandma3.md](docs/grandma3.md) — step-by-step grandMA3 OSC setup
- [docs/protocols.md](docs/protocols.md) — REST + WebSocket reference
- [docs/releasing.md](docs/releasing.md) — versioning, CI releases, CGO per OS
- [CONTRIBUTING.md](CONTRIBUTING.md) · [AGENTS.md](AGENTS.md) (agent/AI-assistant guide)

## Repo layout (short)

```
cmd/showbridge/        CLI entry (serve, config init, midi list|monitor, version)
internal/core/         domain model: Source/Target interfaces, registry, conductor
internal/config/       YAML config model + validation
internal/sources/midi/ MIDI connector (RtMidi, CGO-gated) + board profiles
internal/targets/osc/  OSC/UDP target
internal/server/       REST + WebSocket hub + embedded SPA
web/                   Svelte 5 (runes) UI, SvelteKit SPA → embedded in backend
.github/workflows/     ci + release pipelines
```

*“showbridge” is a working title — rename early via
`rg -l "yourorg/showbridge" | xargs sed -i 's#github.com/yourorg/showbridge#…#g'`.*

## License

MIT (see [LICENSE](LICENSE)).
