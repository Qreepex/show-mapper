# MIDI devices: drivers, profiles, adding boards

## Q: Do I need a driver per MIDI board?

**No.** The APC family (APC mini, mini mk2, APC 20, APC 40) and virtually all
USB MIDI controllers are **USB-MIDI class-compliant**. The operating system
exposes them through its built-in MIDI stack:

| OS | Stack used by show-mapper (via RtMidi) | Notes |
| --- | --- | --- |
| Windows | WinMM (Windows Multimedia) | Zero install. **Caveat:** WinMM allows only **one process** to open a port at a time - close DAWs/other tools if a device seems "dead". |
| macOS | CoreMIDI | Zero install. |
| Linux | ALSA sequencer | Kernel handles everything; your user needs access to `/dev/snd/*` - typically add yourself to the `audio` group (`sudo usermod -aG audio $USER`, re-login). |

What each board **does** need inside show-mapper is a **profile**: the map of
hardware notes/CCs to named controls (plus LED capabilities). Built-in
profiles ship for confirmed boards; anything else can be described as a
**custom board** in config or the UI - no recompilation, no drivers.

## Built-in profiles

### `mpk-mini-mk3` — Akai MPK mini MK3 (✅ VERIFIED on hardware)

Measured with `show-mapper midi monitor` on a real unit (factory program):

- **Touch pads**: CC **32–39**, press 127 / release 0 — threshold-decoded to
  pressed/released → `pad-a-1..8`.
- **Knobs K1–K8**: CC **16–23** (absolute) → `knob-1..8`.
- **Joystick**: X = CC **80** → `joystick-x`, Y = CC **81** → `joystick-y`.
- **Keyboard** (25 keys): notes starting at **48**, intentionally unmapped —
  they arrive as raw `note:NN` to bind ad-hoc.
- **Sustain pedal**: CC **64** → raw `cc:64`, bind with `trigger: value`.
- Bank/Arp/Octave keys are internal functions (no MIDI output).
- Pad LEDs are not host-addressable → no LED feedback.

### `apc-mini-mk2` — verified against the official Akai protocol doc

Source: "APC mini mk2 Communications Protocol v1.0" (Akai/inMusic PDF).

- **Pads** (8×8 RGB): notes **0–63**, note = row·8 + col, **row 0 = bottom
  left** (`pad-<row>-<col>`). Modes send on different channels (we accept all).
- **Track buttons** (bottom row): notes **100–107** → `button-track-1..8`, red LEDs.
- **Scene Launch** (right column): notes **112–119** → `button-scene-1..8`, green LEDs.
- **Shift**: note **122** → `button-shift`, no LED.
- **Faders 1–8**: CC **48–55** → `fader-1..8`; **Master**: CC 56 → `fader-master`.
- **Handshake**: on connect we send the "Introduction" SysEx
  (`F0 47 7F 4F 60 00 04 00 <verH><verL><bugfix> F7`).
- **LEDs**:
  - Pads: Note On where **channel = behavior** (ch 6 = solid 100%, 7–10 = pulse
    speeds, 11–15 = blink speeds) and **velocity = one of 128 palette colors**.
    Palette names exposed in config: white 3, red 5, orange 9, yellow 13,
    green 21, cyan 36, blue 45, purple 48, pink 52.
  - Buttons: Note On ch 0, velocity 0/1/2 = off/on/blink (fixed hardware colors).
  - Bulk update + full custom RGB via SysEx (`0x24` message) exist in the
    protocol - candidates for a later "paint" feature.

### `apc-mini` (mk1) - community-documented, **verify before showtime**

Pads 0–63; bottom buttons 64–71; right column 82–89; shift 122; faders
CC 48–56. LED velocity colors (0=off, 1=green, 2=green blink, 3=red,
4=red blink, 5=yellow, 6=yellow blink). Marked `TODO(hardware)` in code -
please verify with `show-mapper midi monitor` and report corrections.

### `apc-20` / `apc-40` / `apc-40-mk2`

**Not contributed yet** - they're class-compliant and will work the moment a
profile exists (own the hardware? → CONTRIBUTING.md; verify with
`midi monitor`, add a constructor in `internal/sources/midi/profile_akai.go`).
Until then they can already run as **custom boards** (below).

## Custom boards (user-defined)

Describe any board in `show-mapper.yaml`:

```yaml
profiles:
  - id: my-nano
    type: midi
    name: Korg nanoKontrol2 (example)
    match: ["nanoKONTROL"]         # port-name substrings → auto-detect
    led: { style: onOff }          # none | onOff | velocity | apc2-rgb
    controls:
      - { id: fader-1, kind: fader,  label: "Fader 1", cc: 0 }
      - { id: knopf-1, kind: button, label: "S 1",      note: 32, hasLED: true }
```

LED styles: `onOff` (NoteOn 127 / off 0, `onVelocity` adjustable),
`velocity` (color name → velocity table, `colors:` configurable; defaults to
the mk1 scheme), `apc2-rgb` (mk2 native behavior), `none`.
`ledNote` on a control overrides the LED note when it differs from the input note.

**Discovery workflow ("poor man's MIDI learn"):**

1. Open Dashboard → press buttons / move faders.
2. Unmapped controls appear as `note:36` / `cc:10` in the live ticker.
3. Either bind them directly (`control: note:36`) or copy them into a custom
   profile (Settings → Custom boards) for permanent labels/LEDs.
4. CLI equivalent: `show-mapper midi list` + `show-mapper midi monitor <substr>`.

## Layout conventions

Control ids are stable, kebab-case, and hardware-oriented. Pads are
zero-based `pad-<row>-<col>` with **row 0 at the bottom** (following Akai's
note numbering - if your board numbers from the top, encode that in its
profile). Everything UI-facing displays `label` instead.

## Hot-plug & port numbering

RtMidi enumerates ports at open-time; the conductor re-tries failed connects
every 5 s, so plugging in a board "just works" shortly after. Index-based port
identity may shift when devices are added/removed - the config therefore
matches **by name substring**, which survives re-enumeration.
