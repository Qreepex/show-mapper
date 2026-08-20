# show-mapper - Architecture & Build Plan

This is the system design. Read it before extending the code. Related:
[midi-devices.md](midi-devices.md) · [protocols.md](protocols.md) ·
[releasing.md](releasing.md) · module docs live in their module dirs
(e.g. [../internal/helpers/gma3/README.md](../internal/helpers/gma3/README.md))
---

## 1. What & why

`show-mapper` bridges **control surfaces** (MIDI boards today; Elgato Stream
Deck next; more later) to **live-event software** (grandMA3 via OSC today;
ArtNet/sACN and timecode later). It runs as a single binary on a show laptop
(Windows/macOS/Linux), hosts a local web UI for configuration, and streams
all activity to the UI in realtime over WebSocket.

Design goals, in order:

0. **Core stays generic.** The top level knows *sources*, *targets*,
   *bindings* — nothing about grandMA3, Akai, or Elgato. Every device- or
   console-specific thing (code + docs) lives inside a module directory.
   **Module completeness rule:** the binary runs correctly with any subset —
   even zero — of modules compiled in, and with any mix at runtime (e.g.
   only `midi`, only `gma3`, or everything together). Modules register via
   `init()`; inclusion is an import list in `cmd/show-mapper/main.go`
   (**add a module = add one import; remove = delete one line**). We
   deliberately use compile-time inclusion instead of dynamic
   `.so`-style plugins: Go plugins don't work on Windows and break static
   single-binary distribution. The compiled-in inventory is visible at
   `/api/meta` (sourceTypes/targetTypes + presets per helper module); a
   config referencing a *not*-compiled module type degrades that one
   instance to an error status — everything else keeps running
   (tests: internal/core/modules_test.go).
1. **Modularity** - new sources and new targets are plugins implementing tiny
   interfaces, discovered by registries; the UI learns about them via
   `/api/meta`. No config schema surgery per connector.
2. **Boring reliability** - show context: simple, single binary, no external
   services, explicit retry behavior, nothing panics because a USB cable fell out.
3. **Single-file distribution** - frontend embedded via `go:embed`; one config
   file; zero installers. Portable: extract-and-run on Windows, macOS, Linux;
   amd64 + arm64.
4. **Hackable** - plain Go, no framework magic; Svelte 5 SPA you can reason
   about; frontend/backend wire types are generated, not hand-mirrored.

Non-goals (v1): multi-user auth, distributed operation, playlists/cue stacks,
MIDI routing between apps.

---

## 2. Big picture

```
            USB (class-compliant, no vendor drivers)                show network
┌─────────────────────┐                      ┌──────────────────────────────────────┐
│  Akai APC mini mk2  │                      │                                      │      ┌─────────────────────┐
│  custom MIDI boards │── Source events ──►  │            show-mapper                │─────►│  grandMA3 (OSC/UDP) │
│  Stream Deck (soon) │                      │  ┌────────────┐   ┌───────────────┐  │      └─────────────────────┘
└─────────────────────┘                      │  │  sources/  │──►│  conductor    │  │      ┌─ ArtNet/sACN (soon) │
                                             │  │  midi      │   │  (dispatcher) │  │────X │  timecode (soon)    │
┌─────────────────────┐                      │  │  streamdeck│   └──┬─────────▲──┘  │      └─────────────────────┘
│  Browser (config UI)│◄── WS + REST ───────►│  └────────────┘      │  config  │    │
└─────────────────────┘                      │  ┌────────────┐   ┌──▼──────────┴──┐  │
                                             │  │  targets/  │◄──│ core: events →  │  │
                                             │  │  osc, ...  │   │      actions   │  │
                                             │  └────────────┘   └────────────────┘  │
                                             └──────────────────────────────────────┘
```

**Dataflow (press):** board → RtMidi callback → source decodes → `core.Event`
→ conductor matches bindings (trigger/mode, hold timers, toggle state) →
`core.Action` → target sends (UDP packet) → everything broadcast to the UI
ticker via `core.Sink` incl. failures.

**Feedback path (current):** toggle bindings drive board LEDs via the optional
`core.FeedbackSink` interface (`ControlFeedback{ LED | Text | Icon }`). MIDI
boards use the LED part; display-based surfaces (Stream Deck) will use
Text/Icon. **Planned:** target-side feedback (e.g. MA3 OSC output → LED),
so board state can mirror console state.

---

## 3. Core domain model (`internal/core`)

| Concept | Type | Meaning |
| --- | --- | --- |
| Event | `core.Event{SourceID, Control, Kind, Value0..1, Raw, When}` | something happened on a control. Kinds: `pressed`, `released`, `value` |
| Action | `core.Action{BindingID, TargetID, Kind, Address, Args}` | resolved send-order to a target. Targets see **only** actions |
| Source | interface `Connect/Close/Events/Status/ID/Type/Controls` | event producer; `Events()` is a buffered channel owned by the source |
| Target | interface `Connect/Send/Close/Status/ID/Type` | action consumer |
| Profile | connector-defined device layout | maps hardware addressing ↔ stable control ids (e.g. note 7 → `pad-0-7`) + LED capabilities |
| Binding | config row | `source/control/trigger → target/action` (+ mode, holdMs, led) |
| FeedbackSink | optional source interface | visual state for controls (LED now; key LCDs soon) |
| Conductor | `core.Conductor` | builds instances from config, connects them (retry loop), dispatches, hot-reloads (`Reload`) |
| Sink | `core.Sink` | broadcast abstraction → `server.Hub` → WebSocket |
| Inspector | optional per-source-type func | enumerates hardware for the UI (`GET /api/sources/<type>/inspect`) |

**Trigger semantics:** `pressed` fires on button-down — and with
`mode: momentary` also delivers the configured release side on button-up
(release command/value or a preset's release payload). `released` is a
standalone release trigger (falls back to the press payload when no
release-specific value is set). `hold` fires after `holdMs` of continuous
press (cancelled by early release); `value` fires on analog movement.

**Modes:** `momentary` (default: press → OnPress, release → OnRelease-if-set),
`toggle` (press flips state; on → OnPress + LED, off → OnRelease + LED off).

**Action kinds:** `command` (string payload, e.g. MA3 syntax), `value`
(fixed numbers on press/release), `fader` (source value scaled into a range),
`preset` (helper-module template + params, resolved at dispatch),
serialization `int` (default) or `float` per binding.

Error philosophy: config structural problems → rejected with a message list;
instance-level problems (missing device, bad IP) → that instance shows
`error` status and retries every 5 s; permanent problems (e.g. build without
CGO) → marked error, no retry (`core.PermanentError`).

---

## 4. Configuration model

One YAML file (`show-mapper.yaml`; probe order: `$SHOWMAPPER_CONFIG` →
`./show-mapper.yaml` (working dir) → `show-mapper.yaml` in the **binary's
directory** (portable double-click runs) → `$UserConfigDir/show-mapper/config.yaml`;
`-config` flag overrides). **If none exists, a minimal starter config
(version + http defaults, zero connectors) is auto-created** — at
`PreferredCreatePath()` (first writable of env var → CWD → exe dir →
user-config dir), so first runs go straight to the web UI. Edited by hand or
via the UI; saving through the UI validates, persists atomically (tmp+rename,
`.bak` kept), broadcasts `config.updated`, and hot-reloads all connectors.

Sections (full annotated version: [../show-mapper.example.yaml](../show-mapper.example.yaml)):

- `http.listen` - UI/API bind (default `127.0.0.1:8484`, **no auth yet**).
- `profiles[]` - **user-defined device profiles** ("custom boards"). Built-ins
  ship in code; this is where users describe any other hardware. Each entry
  has a connector `type` (`midi`, later `streamdeck`) + match patterns +
  per-control addressing + LED style.
- `sources[]` - `{id, type, profile?, options{...type-specific...}}`.
- `targets[]` - `{id, type, options{...}}`.
- `bindings[]` - see §3; `id` optional (derived), control ids come from the
  source's profile (or raw discoveries like `note:36` / `cc:10`, emitted by
  unmapped controls - the poor-man's MIDI learn).

Validation rules are centralized in `internal/config` (`Validate`) and the
lists exposed via `/api/meta` keep the UI schema-agnostic.

**Config lifecycle:** one portable YAML - copy it machine→machine as-is.
Every apply path validates → persists atomically → hot-reloads connectors:
(a) UI save (`PUT /api/config`), (b) full YAML import/export via UI
(`POST /api/config/import`, `GET /api/config/export`), (b2) **per-section**
import/export — bindings, sources, targets and profiles are individually
savable/exportable/importable
(`GET /api/config/export/sections/{section}`, `POST /api/config/import/section`,
upsert-by-id or replace) — and (c) hand edits of the
file itself (fsnotify watcher, debounced, de-duplicated against our own
saves). Saved YAML is normalized to 2-space indentation — same style as the
examples — so hand-written entries never fight the writer’s formatting.

---

## 5. Connector model

### 5.1 Registries

`core.RegisterSource(type, TypeInfo, factory)` /
`core.RegisterTarget(type, TypeInfo, factory)` - called from each connector
package's `init()`; `cmd/show-mapper/main.go` imports connectors explicitly
(listed there). `TypeInfo` includes the **option field schema**
(`[]FieldSpec`) so the Settings UI renders forms it knows nothing about.

### 5.1b Helper modules, action presets, and target-typed actions

Connectors stay 100% generic; *ecosystem conveniences* (like grandMA3's
Go/Flash/Temp/On/Off vocabulary AND its `gma3` target type) ship as **helper
modules** under `internal/helpers/<name>`. A helper module may register:
- **action presets** (`core.RegisterActionPreset`) — 
  resolvable templates with parameter fields (`FieldSpec`s), scoped per target
  type (`TargetTypes`), e.g. only offered when a binding's target is `gma3`.
- a **target type** from inside the module (`core.RegisterTarget("gma3", …)`)
  — self-contained transport code; the generic connectors never name vendors.

Presets are stored in bindings (`action: {type: preset, preset: id, params}`)
and 
**resolved at dispatch time** (`core/binding.go`) — the resolved result flows
through the same press/release/toggle/hold/value machinery as hand-written
actions. The UI adapts: bindings against such targets render the preset's
fields instead of raw action forms ("generic / raw" stays available).
Project rule: all module-specific code *and documentation* live inside the
module dir (example: `internal/helpers/gma3/README.md`).

### 5.2 Adding a source - checklist (worked example: **Stream Deck**, the planned next step)

1. New package `internal/sources/streamdeck` implementing `core.Source`.
   Key events (`pressed`/`released`) fit today; Stream Deck Plus dials →
   `value` events; pedal → pressed/released.
2. Hardware: Stream Decks are **USB HID** devices (not MIDI, not
   class-audio). Go access via hidapi bindings (e.g. `sstallion/go-hid` /
   `karalabe/hid` or the `magicmonkey/go-streamdeck` helper lib - evaluate
   maintenance + license, then pick). These need CGO → **same pattern as
   MIDI**: `hid_cgo.go` (+ real impl) and `hid_nocgo.go` (stub returning
   `core.PermanentError`). CI matrix already installs toolchains per OS;
   Linux needs a udev rule (docs).
3. Profiles per model (`streamdeck-mini|mk2|xl|pedal|plus`): key count/grid +
   image size (mk2: 15 keys @72×72). Same `Profile` concept as MIDI; the
   config `profiles:` section already supports user-defined ones by `type`.
4. Feedback: implement `core.FeedbackSink` - toggle state can later render
   `Text`/`Icon` onto key LCDs (the interface fields already exist);
   conductor needs no changes.
5. Discovery: `core.RegisterInspector("streamdeck", ...)` enumerating
   attached decks → `GET /api/sources/streamdeck/inspect` + UI picker comes free.
6. Tests, docs tables, README status row, example YAML snippet.

### 5.3 Adding a target - checklist (worked example: ArtNet/sACN)

1. `internal/targets/artnet|sacn` implementing `core.Target`; map
   `core.Action` to channel/value semantics. Likely fields to add to
   `config.ActionConfig`: universe/channel addressing - see TODO note in the
   file; bumping `ActionTypes` flows into UI automatically via `/api/meta`.
2. Library candidates: `github.com/jsimonetti/go-artnet`,
   `github.com/Hundemeier/go-sacn` (evaluate freshness; both pure Go).
3. Options schema must cover **NIC/interface choice** (bind IP), broadcast vs
   unicast target, universe/net/subnet, priority (sACN), rate limiting.
4. Same docs/tests/registration steps as 5.2.

### 5.4 Network plan (interfaces are user-configurable)

- **OSC (implemented):** per-target `host`, `port`, `prefix` **and
  `localAddress`** — the local IPv4/NIC the outgoing UDP socket binds to.
  Machines with multiple NICs simply run several target instances with
  different `localAddress` values — simultaneously. NIC discovery helper:
  `GET /api/system/interfaces` (+ Settings UI listing).
- **ArtNet/sACN (plan):** per-target NIC bind, universe/net/subnet, unicast
  vs broadcast/multicast, priority (sACN), rate limiting — same instance
  pattern as OSC.
- **Firewall cheat sheet:** OSC commonly UDP 8000/9000, Art-Net UDP 6454,
  sACN UDP 5568, show-mapper UI TCP 8484.

### 5.5 MIDI specifics

Driver question, answered fully in [midi-devices.md](midi-devices.md):
**no vendor drivers** - class-compliant devices + OS stacks (WinMM/CoreMIDI/
ALSA) via the bundled RtMidi binding. CGO is required; the build system
handles it (`!cgo` stub keeps dev/CI simple). Per-board support = *profiles*,
either built-in (verified with hardware) or user-defined custom boards from
config/UI - the same runtime `Profile` object is built from both sources.

---

## 6. Frontend architecture

- **Wire types are generated, never hand-mirrored** (`tygo.yaml`,
  `make types` → `web/src/lib/generated/*`). Hand-written UI-only helpers
  live in `web/src/lib/types.ts`. CI fails if generated files went stale.
- **Svelte 5 SPA** (runes only; enforced by project policy + svelte-check),
  SvelteKit with `@sveltejs/adapter-static`, `ssr=false`, `prerender=true`,
  fallback `200.html`. Build output lands in `internal/server/dist/` and is
  embedded by `go:embed`. Committed placeholders let plain `go build` work
  on a fresh clone; CI builds the SPA first.
- **Realtime state**: `web/src/lib/ws.svelte.ts` - a runes class mirroring
  connection state, snapshot, connector statuses and the event ticker;
  auto-reconnect with backoff.
- **Routes**: `/` dashboard (connectors, MIDI discovery, live ticker),
  `/mappings` (bindings editor incl. action presets + control picker from
  profile metadata with free-text raw `note:N`/`cc:N` escapes), `/settings`
  (HTTP, sources, targets, custom board editor, config import/export).
- Dev loop: `vite dev` on :5173 proxies `/api` + `/ws` → :8484 - no CORS,
  no origin juggling.

---

## 7. Realtime protocol (summary)

Single WS endpoint `/ws`, JSON `Envelope{type, ts, data}`; dot-namespaced
types: `state.snapshot` (on connect), `source.event`, `target.action`,
`connector.status`, `config.updated`; inbound reserved as `client.*`.
REST: `/api/health|meta|config|state` · config `export|import` ·
`POST /api/presets/resolve` · `/api/sources/{type}/inspect`.
Full reference with payloads: [protocols.md](protocols.md).

---

## 8. Build & release strategy (summary)

CGO is unavoidable for real device I/O (RtMidi C++). Therefore:

- **Dev/CI fast path**: `CGO_ENABLED=0` compiles everything (stub MIDI),
  tests run everywhere, no toolchain needed.
- **Releases**: native builders per OS/arch in GitHub Actions
  (no cross-compilation of C++): linux/amd64 + linux/arm64 (gcc+ALSA headers),
  windows/amd64 (MinGW gcc), darwin/amd64 + darwin/arm64 (clang). Version via
  git tag → ldflags into `internal/version`. The same assets feed the built-in
  self-updater (checksums validated). Details + platform notes:
  [releasing.md](releasing.md).

---

## 9. Security model (v1)

Trusted-show-network assumption. The HTTP server defaults to localhost; when
bound to LAN it is **unauthenticated** - firewalls/NIC choice are the
boundary (documented for users). OSC/ArtNet/sACN are unauthenticated
protocols by nature. Auth (session token), TLS, and per-IP rules are roadmap
items, deliberately flagged rather than done half-heartedly.

---

## 10. Naming conventions

| Thing | Convention | Examples |
| --- | --- | --- |
| Module path | `github.com/Qreepex/show-mapper` (placeholder, rename early) | - |
| Binary / config file | `show-mapper` / `show-mapper.yaml` | |
| Go packages | lowercase singular; connectors `sources/<type>`, `targets/<type>` | `internal/sources/midi` |
| Connector types | lowercase keyword | `midi`, `osc`, `streamdeck`, `artnet`, `sacn`, `timecode-*` |
| Instance ids (sources/targets/profiles/bindings) | `[a-z0-9][a-z0-9-]*` | `wing-left`, `ma3`, `apc-mini-mk2` |
| Control ids | profile-defined kebab-case; pads are zero-based `pad-<row>-<col>` (row 0 = bottom, Akai note numbering) | `pad-0-3`, `fader-1`, `button-scene-2`, `fader-master`; raw discoveries `note:36`, `cc:10` |
| Triggers / modes / actions | fixed enums | `pressed | released | hold | value`,`momentary | toggle`,`command | value | fader` |
| WS message types | `<domain>.<action>` | `state.snapshot`, `source.event`, `target.action`, `connector.status`, `config.updated`; client→server reserved `client.*` |
| REST | `/api/<noun>` (+ verb for inspect) | `GET/PUT /api/config`, `GET /api/sources/midi/inspect` |
| Git tags / semver | `vMAJOR.MINOR.PATCH`; pre-1.0 minors = features, patches = fixes | `v0.1.0` |
| Release artifacts | `show-mapper_<ver>_<os>_<arch>.{zip,tar.gz}` + `checksums.txt` | `show-mapper_0.1.0_windows_amd64.zip` |
| Commits | conventional commits, scopes: config, conductor, midi, osc, streamdeck, server, web, ci, docs, deps | `feat(midi): …` |
| Code TODOs | `TODO(hardware)` = needs physical verification; otherwise `TODO(scope)` | |

---

## 11. Repo layout

```
├─ cmd/show-mapper/          main + subcommands + embedded example template
├─ internal/
│  ├─ core/                 types, interfaces, registries, conductor, binding resolution
│  ├─ config/               config schema, validation, load/save
│  ├─ sources/midi/         HW abstraction (cgo/nocgo), source impl, profiles (builtin+custom)
│  ├─ targets/osc/          OSC/UDP target
│  ├─ server/               REST, WS hub, SPA embedding (+ dist/ placeholders)
│  └─ version/              ldflags-stamped build info
├─ web/                     Svelte 5 SPA (adapter-static → internal/server/dist)
├─ docs/                    this file, midi-devices, grandma3, protocols, releasing
├─ .github/workflows/       ci.yml, release.yml
├─ Makefile, AGENTS.md, CONTRIBUTING.md, README.md, LICENSE,
│  show-mapper.example.yaml, .golangci.yml, .editorconfig
```

---

## 12. Roadmap

| Milestone | Content |
| --- | --- |
| **v0.1** (this scaffold) | MIDI sources (built-in + custom boards), OSC target + grandMA3 presets, WS UI, config import/export + disk hot-reload, self-update checker, CI/CD |
| v0.2 | Stream Deck source (HID), MIDI-learn→custom-profile wizard in UI, OSC input for LED feedback |
| v0.3 | ArtNet + sACN targets, NIC pickers, unicast/multicast |
| v0.4 | Timecode (MTC in/out, LTC via audio, ArtNet timecode), master clock distribution |
| v0.5 | Page/layer system for binding banks, macros (multi-action bindings), auth/TLS for LAN UI |
