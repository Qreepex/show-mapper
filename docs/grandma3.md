# grandMA3 ← showbridge (OSC)

showbridge's `osc` target speaks plain OSC/UDP to grandMA3 (console or onPC).
This page wires both sides.

## Console-side setup (Menu → In & Out → OSC)

1. Set **Preferred IP** / pick the right **Interface** (the NIC facing the
   showbridge machine).
2. Add/enable an OSC row:
   - **Mode:** UDP · **Port:** e.g. `8000` (MA3 uses this port for *both*
     directions — if you also want MA3→showbridge feedback later, send to
     showbridge on a second row).
   - **Receive:** Yes · **Receive Command:** Yes (only if you use
     `/cmd` actions).
   - **Prefix** (optional): a token like `lights` — must equal showbridge's
     `options.prefix` (no slashes). Useful to filter receivers.
   - The **Page / Fader / ExecutorKnob / Key** columns define which address
     tokens MA3 routes where — defaults match the addresses below
     (**case-sensitive**). **FaderRange** maps incoming numbers to 0–100 %.

3. Firewall/Windows: allow inbound UDP `<port>` to onPC.

## What showbridge can send

| Action type | Address example | Payload | Effect in MA3 |
|---|---|---|---|
| `command` | `/cmd` | string, e.g. `Go Executor 1.201` | runs MA3 keyword syntax (needs *Receive Command*) |
| `value`   | `/Page1/Key201` | int `1` / `0` | press / release executor 201's key on page 1 (0 = released, >0 = pressed) |
| `fader`   | `/Page1/Fader201` | int in your `range` (default 0..100) | executor 201's fader |
| `value`   | `/Page1/ExecutorKnob201` | int | mini encoder of executor 201 |

Notes from the official docs (help.malighting.com, grandMA3 2.x):

- Message form: `/[Address],[type],[value]`; types used here are `i` (int32),
  `f` (float32), `s` (string); `T`/`F` are accepted as 1/0.
- **Executors require a page**: always address `/PageX/...` (page names work too).
- **No OSC bundles** on the receiving side — showbridge sends single messages.
- Executor addressing by name works as well (e.g. `/PageBlue/FaderUp`).
- To *test the wiring backwards*, the console itself can send OSC:
  `SendOSC 1 "/Page1/Fader201,i,50"` (→ into a listener like Protokoll or
  `osc调试` tool on the laptop).

## showbridge-side config

```yaml
targets:
  - id: ma3
    type: osc
    options:
      host: 192.168.1.100   # console/onPC IP
      port: 8000            # the OSC row's Port
      prefix: ""            # the OSC row's Prefix (if any)

bindings:
  - source: wing
    control: pad-0-0
    trigger: pressed
    mode: toggle
    target: ma3
    action:
      type: value
      address: /Page1/Key201
      pressValue: 1
      releaseValue: 0
```

Sanity chain when nothing moves: Dashboard shows `target.action ok:true` →
MA3 System Monitor (EchoInput on the OSC row) shows packets → row's
Receive/Receive Command flags → address case-sensitivity → FaderRange for
fader scaling.

**Feedback direction (planned v0.2):** enable *Send* on an OSC row pointing
back to the laptop; showbridge will then light board LEDs from console state.
