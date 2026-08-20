package midi

import "fmt"

// ---------------------------------------------------------------------------
// Akai MPK mini MK3 — VERIFIED on real hardware (factory default program).
//
// Measured with `show-mapper midi monitor`:
//   touch pads        CC 32–39   (press 127 / release 0) → pad-a-1..8
//   knobs (potis)     CC 16–23   (absolute 0..127)       → knob-1..8
//   joystick          CC 80 (X), CC 81 (Y)               → joystick-x/-y
//   keyboard          notes starting at 48 (raw note:NN, intentionally unmapped)
//   sustain pedal     CC 64 (raw cc:64)
//
// Note: older docs/third-party claims assumed pads=notes 36–43/44–51 and
// knobs=CC 70–77 — those were wrong for this unit. Edit freely if your
// program slot differs; the monitor output tells you the truth.
// ---------------------------------------------------------------------------

func mpkMiniMk3() *Profile {
	p := newProfile(
		"mpk-mini-mk3", "Akai MPK mini MK3",
		// Note: also matches "MPK mini mk2" names — generations share the naming.
		[]string{"MPK mini"},
		ledNone{}, nil,
	)

	// 8 touch pads: CC 32–39, press=127/release=0 (threshold-decoded by the source).
	for i := uint8(1); i <= 8; i++ {
		p.addButtonCC(31+i, fmt.Sprintf("pad-a-%d", i), fmt.Sprintf("Pad A%d", i), false)
	}

	// Knobs K1–K8: CC 16–23 (absolute).
	for i := uint8(1); i <= 8; i++ {
		p.addFader(15+i, fmt.Sprintf("knob-%d", i), fmt.Sprintf("Knob %d", i))
	}

	// Thumbstick: CC 80 (X) / CC 81 (Y).
	p.addFader(80, "joystick-x", "Joystick X")
	p.addFader(81, "joystick-y", "Joystick Y")

	return p
}
