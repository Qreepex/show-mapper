# helper module: grandMA3 (gma3)

This module provides everything grandMA3-flavored in show-mapper:

1. **Target type `gma3`** — OSC/UDP to a grandMA3 console or onPC station
   (`target.go`). Same transport knobs as generic OSC (host/port/prefix/
   localAddress) with MA-appropriate labels/defaults.
2. **Action presets** (`gma3.go`, `gma3.goback`, `gma3.pause`, `gma3.flash`,
   `gma3.temp`, `gma3.on`, `gma3.off`, `gma3.key`, `gma3.fader`, `gma3.cmd`) —
   scoped to `gma3` targets, so the Mappings UI shows them automatically when
   a binding points at a `gma3` instance. Presets are *stored* in the binding
   (`type: preset` + `params`) and resolved at dispatch time — editing params
   re-resolves without touching anything else.

Per project rule, all grandMA3-specific code AND docs live in this directory;
core stays console-agnostic and the binary works with or without this module.

## Console-side setup (Menu → In & Out → OSC)

1. Set **Preferred IP** / pick the right **Interface** (facing this machine).
2. Add/enable an OSC row:
   - **Mode:** UDP · **Port:** e.g. `8000` — must match the target's
     `options.port` (MA3 uses one port for send AND receive per row).
   - **Receive:** Yes · **Receive Command:** Yes (needed for all keyword
     presets: go/goback/pause/temp/on/off/flash/cmd — everything via /cmd).
   - **Prefix** (optional): token like `lights` — must equal `options.prefix`.
   - Address tokens Page/Fader/ExecutorKnob/Key defaults are used by the
     presets (case-sensitive). **FaderRange** maps the fader preset's range
     to 0–100 %.
3. Firewall/Windows: allow inbound UDP `<port>`.

## What presets generate

| preset | OSC message |
|---|---|
| `gma3.go` / `goback` / `pause` / `temp` / `on` / `off` | `/cmd` + string, e.g. `Go+ Page 1.201` |
| `gma3.flash` (momentary) | `/cmd` + string, `Flash On Page 1.201` (press) / `Flash Off Page 1.201` (release) |
| `gma3.cmd` | `/cmd` + your keyword line |
| `gma3.key` (toggle/momentary) | `/Page<P>/Key<E>` int 1 (press) / 0 (release) |
| `gma3.fader` | `/Page<P>/Fader<E>` int mapped through `range` (default 0..100) |

MA facts (official docs, grandMA3 2.x): executors always need a page
(`/PageBlue/…` names work too); addresses case-sensitive & customizable per
row; **no OSC bundles**; `SendOSC 1 "/Page1/Fader201,i,50"` from the console
is a good loopback test.

## Config example

```yaml
targets:
  - id: ma3
    type: gma3
    options: { host: 192.168.1.100, port: 8000 }

bindings:
  - source: wing
    control: pad-0-0
    trigger: pressed
    mode: toggle
    target: ma3
    action: { type: preset, preset: gma3.key, params: { page: "1", executor: "201" } }
    led: { color: green }
```

Troubleshooting: Dashboard shows `target.action ok:true` → MA3 System Monitor
(EchoInput) shows packets → Receive/Receive Command flags → case-sensitivity →
FaderRange mapping. Everything else (raw addresses etc.) is available via the
binding action "— generic / raw OSC —" option, or a plain `type: osc` target.

**Feedback direction (planned):** an OSC row with *Send* back to show-mapper
can later drive board LEDs from console state.
