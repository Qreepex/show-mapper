# helper module: grandMA3 (gma3)

Ready-made **action presets** for MA Lighting grandMA3 consoles/onPC over OSC.
This module registers presets only - it contains no protocol code itself
(that's the generic `internal/targets/osc` module). Per project rule, all
grandMA3-specific code AND documentation live **in this directory**.

Usage in the UI: Mappings → pick a preset (e.g. *grandMA3: Go*) → fill page +
executor → Apply. The preset resolves (`POST /api/presets/resolve`) into a
plain OSC action stored in the binding. Under the hood everything is one of
two OSC shapes below, so hand-written bindings work too.

## Available presets

| preset id | action | resolves to |
| --- | --- | --- |
| `gma3.cmd` | any keyword line | `command @ /cmd`, e.g. `Store Cue 1` |
| `gma3.go` / `goback` / `pause` | Go / Go- / Pause | `command @ /cmd` "… Executor P.E" |
| `gma3.flash` / `gma3.temp` | Flash /Temp tests | `command @ /cmd` |
| `gma3.on` / `gma3.off` | On / Off | `command @ /cmd` |
| `gma3.key` | executor key press/release | `value @ /Page<P>/Key<E>` (1 / 0) - great with `mode: toggle` |
| `gma3.fader` | executor fader | `fader @ /Page<P>/Fader<E>`, range default 0..100 |

## Console-side setup (Menu → In & Out → OSC)

1. Set **Preferred IP** / choose the **Interface** facing the show-mapper machine.
2. An OSC row needs:
   - **Mode:** UDP · **Port:** e.g. `8000` - must match the target's
     `options.port`. MA3 uses one port per row for *both* directions.
   - **Receive:** Yes; **Receive Command:** Yes (needed for `/cmd` presets).
   - **Prefix** (optional): token like `lights`, no slashes - must equal the
     target's `options.prefix`. Addresses then arrive as `/lights/cmd` etc.
   - Address tokens **Page / Fader / ExecutorKnob / Key** (defaults),
     **FaderRange** (maps incoming range to 0–100 %), **Send/Echo** as needed.
3. Windows/macOS firewall: allow inbound UDP `<port>` to the onPC process.

## Addressing reference

Message form `"[Address],[type],[value]"` - the presets above generate:

| intent | address | payload |
| --- | --- | --- |
| run keyword | `/cmd` | string, e.g. `Go Executor 1.201` |
| executor key | `/Page<P>/Key<E>` | int: `0` released, `>0` pressed |
| executor fader | `/Page<P>/Fader<E>` | int within your range (see FaderRange) |
| executor knob | `/Page<P>/ExecutorKnob<E>` | int |

Facts & limits (from the official MA docs, help.malighting.com, grandMA3 2.x):

- Executors always need a page context (`/PageX/…`, page names allowed too).
- Address tokens are **case-sensitive** and customizable per OSC row.
- **OSC bundles are not supported** by MA3 - single messages only (that's what
  the osc module sends).
- Forward test from the console itself: `SendOSC 1 "/Page1/Fader201,i,50"`.

## show-mapper-side example

```yaml
targets:
  - id: ma3
    type: osc
    options:
      host: 192.168.1.100   # console/onPC IP
      port: 8000            # the OSC row's Port
      prefix: ""

bindings:
  - source: wing
    control: pad-0-0
    trigger: pressed
    mode: toggle
    target: ma3
    action:                 # = what preset "gma3.key" with page=1 exec=201 makes
      type: value
      address: /Page1/Key201
      pressValue: 1
      releaseValue: 0
```

Troubleshooting order: Dashboard shows `target.action ok:true` → MA3 System
Monitor shows the packet (EchoInput) → Receive/Receive Command flags →
address case-sensitivity → FaderRange mapping.

**Feedback direction (planned):** an OSC row with *Send* back to show-mapper
can later drive board LEDs from console state.
