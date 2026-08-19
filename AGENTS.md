# AGENTS.md — working agreement for AI agents & contributors in this repo

Read this before touching code. Deeper background lives in `docs/architecture.md`.

## What this is

`showbridge`: a Go backend that bridges control surfaces (MIDI boards now,
Stream Decks next) to show-software targets (grandMA3 over OSC now; ArtNet/
sACN + timecode later), with an embedded Svelte 5 web UI (SPA) for realtime
configuration. Everything is config-file driven; the UI edits the same file
via `PUT /api/config`.

## Commands (Makefile is the source of truth)

| Task | Command | Notes |
|---|---|---|
| Build (full, MIDI) | `make build` | needs C/C++ toolchain (RtMidi is C++) |
| Build (fast, no MIDI) | `make build-nocgo` | `CGO_ENABLED=0`, works everywhere |
| Run backend | `make run` | serves :8080; placeholder UI until `make web` |
| Frontend build | `make web` | outputs into `internal/server/dist/` (embedded) |
| Frontend dev | `make web-dev` + `make run` | vite on :5173 proxied to :8080 |
| Tests + vet | `make test` / `make vet` | always `CGO_ENABLED=0` for tests |
| Full local CI | `make check` | vet, tests, svelte-check, builds |
| Lint | `make lint` | golangci-lint (config: `.golangci.yml`) |

CI runs: `go vet`, `go test`, `go build` (CGO off), `npm run check`, `npm run build`, golangci-lint. **Keep `make check` green.**

## Architecture invariants (do not break)

1. **`internal/core` owns the domain model and must not import connectors.**
   Sources/targets register via `core.RegisterSource/RegisterTarget` from their
   package `init()`; `cmd/showbridge/main.go` imports connectors (one blank
   import per connector).
2. **Connector packages are self-contained**: `internal/sources/<type>`,
   `internal/targets/<type>`. No cross-imports between connectors.
3. **Hardware access is build-tagged.** Anything needing a C/C++ stack
   (RtMidi: `driver_cgo.go`) has a `driver_nocgo.go` twin so
   `CGO_ENABLED=0 go build ./... && go test ./...` *always* works. The stub
   returns a **permanent** error (`core.PermanentError`) — the conductor then
   reports instead of retry-looping.
4. **Config is validated twice**: structurally in `config.Validate()` and at
   instance creation (unknown connector/profile → per-instance `error`
   status, never a process crash).
5. **One wire format to the UI**: WS `Envelope{type, ts, data}` with
   dot-namespaced types (`docs/protocols.md`). Broadcast via `core.Sink` only;
   core knows nothing about net/http.
6. **ES: The UI never hardcodes connector knowledge.** It renders from
   `/api/meta` (connector types, option field specs, device profiles,
   enumerations). Adding a connector automatically surfaces in the UI.

## Svelte rules (hard requirements, from global policy)

- **Svelte 5 runes only**: `$state`, `$derived`, `$effect`, `$props()`.
  Forbidden: `$:` statements, `export let`, `on:click` directives (use
  `onclick={...}`), `createEventDispatcher`, `<slot>` for new components
  (use `{#snippet}`/`{@render}`), svelte stores (`writable/derived`) — shared
  state lives in `.svelte.ts` modules with runes (see `web/src/lib/ws.svelte.ts`).
- `$app/state`, not `$app/stores`.
- svelte-check (`npm run check`) must pass with **0 errors/warnings**.

## Go rules

- `gofmt`/`go vet` clean; table-driven tests for mapping logic; keep connector
  code side-effect free outside `Connect()`.
- No new dependency without a short justification in the PR (current set:
  yaml.v3, coder/websocket, go-osc, gomidi's bundled rtmidi binding).
- Version stamping only via `internal/version` ldflags (see docs/releasing.md).

## Common recipes

### Add a target connector (e.g. ArtNet/sACN)
1. Create `internal/targets/<type>/` implementing `core.Target`
   (`ID/Type/Connect/Send/Close/Status`).
2. In `init()`: `core.RegisterTarget("<type>", core.TypeInfo{...options schema...}, NewTarget)`.
3. Blank-import it in `cmd/showbridge/main.go`.
4. Decide how `core.Action` maps to your protocol (extend `ActionConfig` in
   `internal/config` if you need new semantics — update `Validate`,
   `ActionTypes`, tests, `web/src/lib/types.ts`, docs).
5. Tests + docs/architecture.md “Connectors” table + README status table.

### Add a source connector (e.g. Elgato Stream Deck) — next planned
See docs/architecture.md#adding-a-source for the full checklist incl.
HID/CGO strategy, profiles, feedback (`core.FeedbackSink` already supports
`Text`/`Icon` for display keys), and the inspector hook
(`core.RegisterInspector` → `GET /api/sources/<type>/inspect`).

### Add / fix a built-in MIDI board profile
Follow docs/midi-devices.md#adding-a-new-device — **verify against hardware**
with `showbridge midi monitor` before merging; mark unverified data with
`TODO(hardware)` like `apcMini()` does.

### Change the WS/REST contract
Update `docs/protocols.md` + `web/src/lib/types.ts` in the same PR.

## Naming conventions (summary; full list in architecture.md §Naming)

- Instance/profile/binding ids: `[a-z0-9][a-z0-9-]*` (enforced by config).
- Connector type names: lowercase (`midi`, `osc`, `artnet`, `streamdeck`).
- WS types: `<domain>.<action>` (`source.event`, `target.action`,
  `connector.status`, `state.snapshot`, `config.updated`, `client.*` reserved).
- Tags: `v<semver>`; release artifacts `showbridge_<ver>_<os>_<arch>.{zip,tar.gz}`.
- Commits: conventional (`feat(scope): …`); scopes: midi|osc|core|server|web|ci|docs.

## Gotchas collection

- `internal/server/dist/index.html` + `200.html` are committed placeholders by
  design (fresh-clone `go build` must embed *something*). Real web builds
  overwrite them — **do not commit web build output** (`.gitignore` handles
  assets; if git tracks a rebuilt placeholder, restore it).
- Windows + WinMM: a MIDI port can be opened by only one process at a time.
  If `showbridge` can't see presses, close other tools (DawControl, DAWs, browsers).
- RtMidi port enumeration is static — hot-plug shows up after the conductor's
  5 s reconnect cycle, not instantly.
- No CGO on your machine? Everything except MIDI hardware works; tests/CI
  default to `CGO_ENABLED=0`.
