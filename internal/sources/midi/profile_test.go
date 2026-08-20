package midi

import (
	"testing"

	"github.com/Qreepex/show-mapper/internal/config"
	"github.com/Qreepex/show-mapper/internal/core"
)

func TestProfileForDeviceOrdering(t *testing.T) {
	// "APC mini" is a substring of the mk2 port name — mk2 must win.
	if p := ProfileForDevice("APC mini mk2"); p == nil || p.ID != "apc-mini-mk2" {
		t.Fatalf("APC mini mk2 → %v", p)
	}
	if p := ProfileForDevice("2- APC mini mk2 [1]"); p == nil || p.ID != "apc-mini-mk2" {
		t.Fatalf("windows-style suffix → %v", p)
	}
	if p := ProfileForDevice("APC mini"); p == nil || p.ID != "apc-mini" {
		t.Fatalf("APC mini → %v", p)
	}
	if p := ProfileForDevice("nanokontrol2"); p != nil {
		t.Fatalf("unknown board should not match, got %v", p.ID)
	}
}

func TestMk2Mapping(t *testing.T) {
	p, ok := ProfileByID("apc-mini-mk2")
	if !ok {
		t.Fatal("apc-mini-mk2 profile missing")
	}

	// pads: note 0 = pad-0-0, note 63 = pad-7-7
	if got := p.Notes[0]; got != "pad-0-0" {
		t.Errorf("note 0 → %q, want pad-0-0", got)
	}
	if got := p.Notes[63]; got != "pad-7-7" {
		t.Errorf("note 63 → %q, want pad-7-7", got)
	}
	// faders: CC 48 = fader-1, CC 56 = fader-master
	if got := p.CCs[48]; got != "fader-1" {
		t.Errorf("cc 48 → %q, want fader-1", got)
	}
	if got := p.CCs[56]; got != "fader-master" {
		t.Errorf("cc 56 → %q, want fader-master", got)
	}
	// shift = note 122, track 1 = 100, scene 8 = 119
	for note, want := range map[uint8]string{122: "button-shift", 100: "button-track-1", 119: "button-scene-8"} {
		if got := p.Notes[note]; got != want {
			t.Errorf("note %d → %q, want %q", note, got, want)
		}
	}
	// round trip: every NoteOf target exists in Notes
	for ctrl, note := range p.NoteOf {
		if _, ok := p.Notes[note]; !ok {
			t.Errorf("NoteOf[%s]=%d has no Notes entry", ctrl, note)
		}
	}
}

func TestMk2LEDMessages(t *testing.T) {
	p, _ := ProfileByID("apc-mini-mk2")

	// pad 7, solid 100% green: {0x96, 7, 21}
	got := p.LED.Messages(7, core.LEDState{Mode: "on", Color: "green"})
	want := [][]byte{{0x96, 7, 21}}
	assertMsgs(t, got, want)

	// pad 7 blink red: channel 13 (blink 1/8), velocity 5
	got = p.LED.Messages(7, core.LEDState{Mode: "blink", Color: "red"})
	assertMsgs(t, got, [][]byte{{0x90 | 13, 7, 5}})

	// pad off: velocity 0
	got = p.LED.Messages(7, core.LEDState{Mode: "off"})
	if len(got) != 1 || got[0][2] != 0 {
		t.Errorf("pad off → %v", got)
	}

	// button (note 100) blink: ch 0, vel 2
	got = p.LED.Messages(100, core.LEDState{Mode: "blink", Color: "red"})
	assertMsgs(t, got, [][]byte{{0x90, 100, 2}})
}

func TestProfileFromConfig(t *testing.T) {
	note, cc := 36, 1
	pc := config.ProfileConfig{
		ID: "my-board", Type: "midi", Name: "My Board", Match: []string{"MYBOARD"},
		LED: config.ProfileLED{Style: "onOff"},
		Controls: []config.ProfileControl{
			{ID: "pad-1", Kind: "pad", Label: "Pad 1", Note: &note, HasLED: true},
			{ID: "fader-1", Kind: "fader", Label: "Fader 1", CC: &cc},
		},
	}
	p, err := ProfileFromConfig(pc)
	if err != nil {
		t.Fatalf("ProfileFromConfig: %v", err)
	}
	if p.Notes[36] != "pad-1" || p.CCs[1] != "fader-1" {
		t.Errorf("maps wrong: notes=%v ccs=%v", p.Notes, p.CCs)
	}
	if _, ok := p.NoteOf["pad-1"]; !ok {
		t.Error("pad-1 should be LED-addressable")
	}
	got := p.LED.Messages(36, core.LEDState{Mode: "on"})
	assertMsgs(t, [][]byte{{0x90, 36, 127}}, got)

	// collision with a built-in id must fail
	pc.ID = "apc-mini-mk2"
	if _, err := ProfileFromConfig(pc); err == nil {
		t.Error("expected collision error for built-in id")
	}
}

func assertMsgs(t *testing.T, a, b [][]byte) {
	t.Helper()
	if len(a) != len(b) {
		t.Fatalf("message count %d ≠ %d (%v vs %v)", len(a), len(b), a, b)
	}
	for i := range a {
		if len(a[i]) != len(b[i]) {
			t.Fatalf("msg %d length %d ≠ %d", i, len(a[i]), len(b[i]))
		}
		for j := range a[i] {
			if a[i][j] != b[i][j] {
				t.Fatalf("msg %d byte %d: %02x ≠ %02x (got % X want % X)", i, j, a[i][j], b[i][j], a, b)
			}
		}
	}
}

func TestMpkMiniMk3Mapping(t *testing.T) {
	p, ok := ProfileByID("mpk-mini-mk3")
	if !ok {
		t.Fatal("mpk-mini-mk3 profile missing")
	}
	for cc, want := range map[uint8]string{32: "pad-a-1", 39: "pad-a-8"} {
		ctrl, ok := p.CCs[cc]
		if !ok || ctrl != want {
			t.Errorf("cc %d → %q, want %q", cc, p.CCs[cc], want)
		}
		cctl, _ := p.ControlByID(ctrl)
		if cctl.Kind != core.ControlButton {
			t.Errorf("pad %q kind %q, want button (CC threshold decode)", ctrl, cctl.Kind)
		}
	}
	if got := p.CCs[16]; got != "knob-1" {
		t.Errorf("cc 16 → %q, want knob-1", got)
	}
	if got := p.CCs[23]; got != "knob-8" {
		t.Errorf("cc 23 → %q, want knob-8", got)
	}
	if got := p.CCs[80]; got != "joystick-x" {
		t.Errorf("cc 80 → %q, want joystick-x", got)
	}
	if got := p.CCs[81]; got != "joystick-y" {
		t.Errorf("cc 81 → %q, want joystick-y", got)
	}
	if p.PitchBend != "" {
		t.Errorf("pitch bend must be unmapped on the mpk, got %q", p.PitchBend)
	}
	if found := ProfileForDevice("MPK mini 3"); found == nil || found.ID != "mpk-mini-mk3" {
		t.Errorf("device auto-detect → %v", found)
	}
}

func TestCCButtonThreshold(t *testing.T) {
	p, ok := ProfileByID("mpk-mini-mk3")
	if !ok {
		t.Fatal("profile missing")
	}
	src := &Source{profile: p, ccEdge: map[uint8]bool{}}

	on1, handled := src.ccButtonEvent(32, "pad-a-1", 127)
	if !handled || on1 == nil || on1.Kind != core.EventPressed {
		t.Fatalf("press edge: %+v handled=%v", on1, handled)
	}
	repeat, _ := src.ccButtonEvent(32, "pad-a-1", 100)
	if repeat != nil {
		t.Fatal("repeat above threshold must be dropped")
	}
	off, _ := src.ccButtonEvent(32, "pad-a-1", 0)
	if off == nil || off.Kind != core.EventReleased {
		t.Fatalf("release edge: %+v", off)
	}
	// analog knob stays analog:
	ev, handled := src.ccButtonEvent(16, "knob-1", 64)
	if handled || ev != nil {
		t.Fatal("knob CC must NOT be threshold-decoded")
	}
}
