package midi

import (
	"fmt"

	"github.com/yourorg/showbridge/internal/core"
)

// ---------------------------------------------------------------------------
// Akai APC mini mk2 — data verified against the official
// "APC mini mk2 Communications Protocol v1.0" (Akai/inMusic).
// See docs/midi-devices.md for the full protocol summary + source link.
// ---------------------------------------------------------------------------

func apcMiniMk2() *Profile {
	// "Introduction" handshake (inner bytes; 0xF0/0xF7 are added on send):
	// manufacturer 0x47 (Akai), device 0x7F, product 0x4F, msg 0x60,
	// len 4: app-id 0x00, version 1.0.0.
	p := newProfile(
		"apc-mini-mk2", "Akai APC mini mk2",
		[]string{"APC mini mk2"},
		ledAPC2{},
		[]byte{0x47, 0x7F, 0x4F, 0x60, 0x00, 0x04, 0x00, 0x01, 0x00, 0x00},
	)

	// 8x8 RGB pad matrix: notes 0x00–0x3F, note 0 = bottom-left, row-major up.
	for n := uint8(0); n < 64; n++ {
		p.addPad(n)
	}

	// Bottom row: Track buttons 1–8, notes 0x64–0x6B (single red LED).
	for i := uint8(1); i <= 8; i++ {
		p.addButton(0x63+i, fmt.Sprintf("button-track-%d", i), fmt.Sprintf("Track %d", i), true)
	}
	// Right column: Scene Launch buttons 1–8, notes 0x70–0x77 (single green LED).
	for i := uint8(1); i <= 8; i++ {
		p.addButton(0x6F+i, fmt.Sprintf("button-scene-%d", i), fmt.Sprintf("Scene %d", i), true)
	}
	// Shift: note 0x7A, no LED.
	p.addButton(0x7A, "button-shift", "Shift", false)

	// Faders 1–8: CC 0x30–0x37, channel 0, port 0. Master fader: CC 0x38.
	for i := uint8(1); i <= 8; i++ {
		p.addFader(0x2F+i, fmt.Sprintf("fader-%d", i), fmt.Sprintf("Fader %d", i))
	}
	p.addFader(0x38, "fader-master", "Master")
	return p
}

// ---------------------------------------------------------------------------
// Akai APC mini (mk1) — community-documented mapping.
// TODO(hardware): verify bottom-row/scene note numbers and LED velocity
// palette against a real device using `showbridge midi monitor`;
// see docs/midi-devices.md#adding-a-new-device.
// ---------------------------------------------------------------------------

func apcMini() *Profile {
	p := newProfile("apc-mini", "Akai APC mini (mk1)", []string{"APC mini"}, ledAPC1{}, nil)

	// 8x8 pad matrix: notes 0x00–0x3F (same numbering as mk2).
	for n := uint8(0); n < 64; n++ {
		p.addPad(n)
	}

	// Bottom row buttons: notes 0x40–0x47. Labels differ between print runs;
	// keep neutral names until verified.
	for i := uint8(1); i <= 8; i++ {
		p.addButton(0x3F+i, fmt.Sprintf("button-bottom-%d", i), fmt.Sprintf("Bottom %d", i), true)
	}
	// Right column (scene launch): notes 0x52–0x59.
	for i := uint8(1); i <= 8; i++ {
		p.addButton(0x51+i, fmt.Sprintf("button-scene-%d", i), fmt.Sprintf("Scene %d", i), true)
	}
	// Shift: note 0x7A, no LED.
	p.addButton(0x7A, "button-shift", "Shift", false)

	// Sliders 1–8: CC 0x30–0x37, 9th slider (master): CC 0x38.
	for i := uint8(1); i <= 8; i++ {
		p.addFader(0x2F+i, fmt.Sprintf("fader-%d", i), fmt.Sprintf("Fader %d", i))
	}
	p.addFader(0x38, "fader-master", "Master")
	return p
}

// ---------------------------------------------------------------------------
// LED backends
// ---------------------------------------------------------------------------

// ledAPC2 implements the APC mini mk2 LED protocol (official Akai doc):
//
//   - RGB pads (notes 0x00–0x3F): Note On, channel selects behavior
//     (0–5 = solid 10–90%, 6 = solid 100%, 7–10 = pulse, 11–15 = blink),
//     velocity selects one of 128 palette colors.
//   - Single-color buttons (0x64–0x6B red, 0x70–0x77 green): Note On on
//     channel 0, velocity 0=off, 1=on, 2=blink (color is hardware-fixed).
type ledAPC2 struct{}

func (ledAPC2) Name() string { return "apc-mini-mk2 (RGB pads + fixed-color buttons)" }

func (ledAPC2) Messages(note uint8, st core.LEDState) [][]byte {
	if note <= 63 {
		if st.Mode == "off" {
			return [][]byte{{0x90 | 6, note, 0x00}} // any solid channel, velocity 0 = off
		}
		ch := uint8(6) // solid, 100% brightness
		switch st.Mode {
		case "blink":
			ch = 13 // blinking 1/8
		case "pulse":
			ch = 9 // pulsing 1/4
		}
		return [][]byte{{0x90 | ch, note, apc2Color(st.Color)}}
	}
	vel := uint8(1) // on
	switch st.Mode {
	case "off":
		vel = 0
	case "blink", "pulse":
		vel = 2 // single LEDs can only blink
	}
	return [][]byte{{0x90, note, vel}}
}

// apc2Color maps friendly names to the mk2's fixed 128-color palette.
// Values are velocity values from the official Akai palette table.
// Unknown/empty colors fall back to green (21).
func apc2Color(name string) uint8 {
	palette := map[string]uint8{
		"white":  3,  // #FFFFFF
		"red":    5,  // #FF0000
		"orange": 9,  // #FF5400
		"yellow": 13, // #FFFF00
		"green":  21, // #00FF00
		"cyan":   36, // #4CC3FF
		"blue":   45, // #0000FF
		"purple": 48, // #874CFF
		"pink":   52, // #FF4CFF
	}
	if v, ok := palette[name]; ok {
		return v
	}
	return 21
}

// ledAPC1 implements the community-documented APC mini (mk1) velocity-color
// scheme. TODO(hardware): verify palette against a real device.
//
//	velocity: 0=off, 1=green, 2=green blink, 3=red, 4=red blink, 5=yellow, 6=yellow blink
type ledAPC1 struct{}

func (ledAPC1) Name() string { return "apc-mini (velocity-color, unverified)" }

func (ledAPC1) Messages(note uint8, st core.LEDState) [][]byte {
	if st.Mode == "off" {
		return [][]byte{{0x90, note, 0}}
	}
	base := uint8(1) // green
	switch st.Color {
	case "red":
		base = 3
	case "yellow", "amber", "orange":
		base = 5
	}
	if st.Mode == "blink" || st.Mode == "pulse" {
		base++ // blink variant = base color + 1 (up to 6)
		if base > 6 {
			base = 6
		}
	}
	return [][]byte{{0x90, note, base}}
}
