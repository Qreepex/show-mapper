package main

// exampleConfig is written by `show-mapper config init`.
// Keep it in sync with show-mapper.example.yaml at the repo root and with
// config struct field names (both are YAML-marshaled from config.Config).
const exampleConfig = `# show-mapper configuration — full annotated version: show-mapper.example.yaml (repo root)
# Docs: docs/architecture.md • MIDI boards: docs/midi-devices.md • grandMA3: internal/helpers/gma3/README.md

version: 1

http:
  # Web UI + API + WebSocket. No auth yet — keep on localhost or a trusted show network.
  listen: 127.0.0.1:8080

# --- Sources: where events come from ----------------------------------------
sources:
  - id: wing                  # your instance name ([a-z0-9-])
    type: midi
    profile: apc-mini-mk2     # built-in board layout; see docs/midi-devices.md
    options:
      device: APC mini mk2    # substring of the OS port name (show-mapper midi list)

# --- Targets: where actions go ----------------------------------------------
targets:
  - id: ma3
    type: osc
    options:
      host: 192.168.1.100     # grandMA3 console/onPC IP — TODO: change me
      port: 8000              # must match the OSC row's Port in MA3 (Menu > In & Out > OSC)
      # prefix: ""            # optional; must match the MA3 OSC row Prefix

# --- Bindings: control -> action --------------------------------------------
bindings:
  # Pad (row 0 = bottom) toggles executor 201 on page 1 using MA3's OSC fader/key addresses.
  - source: wing
    control: pad-0-0
    trigger: pressed
    mode: toggle
    target: ma3
    action:
      type: value
      address: /Page1/Key201     # executor key: 0 = released, >0 = pressed
      pressValue: 1
      releaseValue: 0
    led: { color: green }

  # Send grandMA3 command line input via OSC (/cmd requires "Receive Command" enabled).
  - source: wing
    control: pad-0-1
    trigger: pressed
    target: ma3
    action:
      type: command
      address: /cmd
      command: Go Executor 1.202

  # Hardware fader -> MA3 executor fader (0..100).
  - source: wing
    control: fader-1
    trigger: value
    target: ma3
    action:
      type: fader
      address: /Page1/Fader201
      range: [0, 100]
      valueType: int
`
