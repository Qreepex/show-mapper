package midi

import "strconv"

// ---------------------------------------------------------------------------
// Akai MPK mini MK3 — VERIFIED on real hardware (factory default program).
//
// Measured with `show-mapper midi monitor`:
//   touch pads        Bank A: 16-23 Bank B:32–39   (press 127 / release 0) → pad-a-1..8 / pad-b-1..8
//   CC touch pads 		 Bank A: 16-23 Bank B:32–39   (press 127 / release 0) → pad-a-1..8 / pad-b-1..8
//   knobs (potis)     CC 16–23   (absolute 0..127)       → knob-1..8
//   joystick          CC 80 (X), CC 81 (Y)               → joystick-x/-y
//   keyboard          25 keys, notes 48–72 (C3–C5)       → key-c3 … key-c5
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

	// 8 touch pads: Bank A: 16-23 Bank B: 32–39, press=127/release=0 (threshold-decoded by the source).
	for i := uint8(1); i <= 8; i++ {
		p.addButton(15+i, padID(i, "a"), padLabel(i, "A"), true)
		p.addButtonCC(15+i, padID(i, "a"), padLabel(i, "A"), true)
	}
	for i := uint8(1); i <= 8; i++ {
		p.addButton(31+i, padID(i, "b"), padLabel(i, "B"), true)
		p.addButtonCC(31+i, padID(i, "b"), padLabel(i, "B"), true)
	}

	// Knobs K1–K8: CC 16–23 (absolute).
	for i := uint8(1); i <= 8; i++ {
		p.addFader(15+i, knobID(i), knobLabel(i))
	}

	// Thumbstick: CC 80 (X) / CC 81 (Y).
	p.addFader(80, "joystick-x", "Joystick X")
	p.addFader(81, "joystick-y", "Joystick Y")

	// 25-key keyboard: C3 (note 48) through C5 (note 72).
	// Mapped as momentary buttons (no LED) — usable as a trigger surface
	// when the MPK is used purely for show control, not music.
	for note := uint8(48); note <= 72; note++ {
		id, label := noteName(note)
		p.addButton(note, id, label, false)
	}

	return p
}

// ---------------------------------------------------------------------------
// Naming helpers
// ---------------------------------------------------------------------------

func padID(i uint8, bank string) string    { return "pad-" + bank + "-" + strconv.Itoa(int(i)) }
func padLabel(i uint8, bank string) string { return "Pad " + bank + strconv.Itoa(int(i)) }
func knobID(i uint8) string                { return "knob-" + strconv.Itoa(int(i)) }
func knobLabel(i uint8) string             { return "Knob " + strconv.Itoa(int(i)) }

// noteName returns the control ID and human-readable label for a MIDI note
// number. Sharp notes get "s" in the ID (e.g. "key-cs3") and "♯" in the
// label (e.g. "C♯3") so IDs are URL-safe and labels are readable.
func noteName(note uint8) (id, label string) {
	octave := int(note)/12 - 1
	semitone := int(note) % 12
	names := [12]struct {
		idShort    string
		labelShort string
	}{
		{"c", "C"},   // 0
		{"cs", "C♯"}, // 1
		{"d", "D"},   // 2
		{"ds", "D♯"}, // 3
		{"e", "E"},   // 4
		{"f", "F"},   // 5
		{"fs", "F♯"}, // 6
		{"g", "G"},   // 7
		{"gs", "G♯"}, // 8
		{"a", "A"},   // 9
		{"as", "A♯"}, // 10
		{"h", "H"},   // 11
	}
	n := names[semitone]
	id = "key-" + n.idShort + strconv.Itoa(octave)
	label = n.labelShort + strconv.Itoa(octave)
	return
}
