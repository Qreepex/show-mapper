package midi

import (
	"fmt"

	"github.com/Qreepex/show-mapper/internal/core"
)

// ---------------------------------------------------------------------------
// Akai MPK mini MK3 — factory default program mapping.
//
// Sources: Akai docs/FAQ + joonas.fi reverse-engineering writeup
// (knobs K1–K8 = CC 70–77 absolute 0-127, pads bank A notes 36–43 / bank B
// 44–51, joystick X = pitch bend, joystick Y = CC 1, keys on channel 1,
// pads channel 10). LED feedback on pads is not addressable from the host, so
// hasLED=false everywhere.
//
// The 25 piano keys are intentionally unmapped — they arrive as raw
// note:NN events (usable directly in bindings).
//
// TODO(hardware): verify knobs' absolute-CC mode & pad bank numbering on a
// real unit via `show-mapper midi monitor`; corrections welcome.
// ---------------------------------------------------------------------------

func mpkMiniMk3() *Profile {
	p := newProfile(
		"mpk-mini-mk3", "Akai MPK mini MK3",
		// Note: also matches "MPK mini mk2" names when no mk2 profile exists —
		// both generations share the same default map; verify before shipping.
		[]string{"MPK mini"},
		ledNone{}, nil,
	)

	// 8 pads × 2 banks (bank A/B hardware button switches banks):
	for i := uint8(1); i <= 8; i++ {
		p.addButton(35+i, fmt.Sprintf("pad-a-%d", i), fmt.Sprintf("Pad A%d", i), false)
	}
	for i := uint8(1); i <= 8; i++ {
		p.addButton(43+i, fmt.Sprintf("pad-b-%d", i), fmt.Sprintf("Pad B%d", i), false)
	}

	// KNOB K1–K8: CC 70–77 (absolute 0..127 in the factory program).
	for i := uint8(1); i <= 8; i++ {
		p.addFader(69+i, fmt.Sprintf("knob-%d", i), fmt.Sprintf("Knob %d", i))
	}

	// Thumbstick: X axis => pitch bend, Y axis => CC 1 (modulation).
	p.PitchBend = "joystick-x"
	p.Controls = append(p.Controls,
		core.Control{ID: "joystick-x", Label: "Joystick X (pitch bend)", Kind: core.ControlFader},
		core.Control{ID: "joystick-y", Label: "Joystick Y (modulation)", Kind: core.ControlFader},
	)
	p.CCs[1] = "joystick-y"

	// Sustain pedal (CC 64) arrives as a raw "cc:64" value event.
	return p
}
