package midi

import (
	"fmt"
	"strings"

	"github.com/Qreepex/show-mapper/internal/core"
)

// ---------------------------------------------------------------------------
// Device profiles
//
// A profile maps hardware notes/CCs to meaningful control IDs + labels and
// describes LED capabilities. This is data, not code — new boards are added
// here after verifying note/CC numbers with `show-mapper midi monitor`.
// See docs/midi-devices.md ("Adding a new device").
// ---------------------------------------------------------------------------

// Profile describes one controller family.
type Profile struct {
	ID    string   // registry ID, e.g. "apc-mini-mk2" (referenced as `profile:` in config)
	Name  string   // human name
	Match []string // case-insensitive substrings of the OS port name, for auto-detection

	Controls []core.Control
	Notes    map[uint8]string        // note number -> control ID
	CCs      map[uint8]string        // cc number   -> control ID
	NoteOf   map[string]uint8        // control ID  -> note number (for LED feedback)
	byID     map[string]core.Control // control ID -> control (kind lookup)

	// PitchBend, if non-empty, maps pitch-bend messages to this control ID
	// (e.g. "joystick-x" on the MPK mini mk3).
	PitchBend string

	// IntroSysex, if set, is sent right after connect (inner bytes only,
	// without 0xF0/0xF7). The APC mini mk2 requires this "Introduction"
	// handshake before device-specific traffic.
	IntroSysex []byte

	LED ledBackend
}

// ledBackend translates abstract LED requests into raw MIDI messages.
type ledBackend interface {
	Name() string
	// Messages builds one or more raw MIDI messages (status+data bytes).
	Messages(note uint8, st core.LEDState) [][]byte
}

// profiles holds all registered profiles. ORDER MATTERS for auto-detection:
// "APC mini" is a substring of the "APC mini mk2" port name, so the mk2
// profile must be matched first.
var profiles = []*Profile{apcMiniMk2(), apcMini(), mpkMiniMk3()}

var profileIndex = func() map[string]*Profile {
	m := map[string]*Profile{}
	for _, p := range profiles {
		m[p.ID] = p
	}
	return m
}()

// ProfileByID returns the profile for a config value.
func ProfileByID(id string) (*Profile, bool) {
	p, ok := profileIndex[id]
	return p, ok
}

// ProfileIDs lists all profile IDs (sorted by match order).
func ProfileIDs() []string {
	out := make([]string, 0, len(profiles))
	for _, p := range profiles {
		out = append(out, p.ID)
	}
	return out
}

// ProfileForDevice auto-detects a profile from a port name (first match wins).
func ProfileForDevice(portName string) *Profile {
	lower := strings.ToLower(portName)
	for _, p := range profiles {
		for _, m := range p.Match {
			if strings.Contains(lower, strings.ToLower(m)) {
				return p
			}
		}
	}
	return nil
}

// Profiles returns all profiles for /api/meta.
func Profiles() []core.ProfileSummary {
	out := make([]core.ProfileSummary, 0, len(profiles))
	for _, p := range profiles {
		led := "none"
		if p.LED != nil {
			led = p.LED.Name()
		}
		out = append(out, core.ProfileSummary{ID: p.ID, Name: p.Name, LED: led, Controls: p.Controls})
	}
	return out
}

// ---------------------------------------------------------------------------
// Shared builders
// ---------------------------------------------------------------------------

func newProfile(id, name string, match []string, led ledBackend, intro []byte) *Profile {
	return &Profile{
		ID: id, Name: name, Match: match,
		Notes: map[uint8]string{}, CCs: map[uint8]string{}, NoteOf: map[string]uint8{},
		LED: led, IntroSysex: intro,
	}
}

// addPad registers a matrix pad: note -> "pad-<row>-<col>".
// Akai numbering: note 0 = bottom-left, row-major upward (row = note/8, col = note%8).
func (p *Profile) addPad(note uint8) {
	row, col := int(note)/8, int(note)%8
	id := fmt.Sprintf("pad-%d-%d", row, col)
	c := core.Control{
		ID: id, Kind: core.ControlPad, Row: row, Col: col, HasLED: true,
		Label: fmt.Sprintf("Pad %d/%d", row+1, col+1),
	}
	p.addControl(c)
	p.Notes[note] = id
	p.NoteOf[id] = note
}

func (p *Profile) addButton(note uint8, id, label string, hasLED bool) {
	c := core.Control{ID: id, Kind: core.ControlButton, Label: label, HasLED: hasLED}
	p.addControl(c)
	p.Notes[note] = id
	p.NoteOf[id] = note
}

// addControl appends the control and keeps the by-ID lookup in sync.
// The Controls inventory is unique by control ID: one control may be
// reachable through several message routes (a profile may map both a note
// and a CC to the same pad), but it must appear in Controls only once —
// the UI keys option lists by control ID (Svelte each_key_duplicate
// otherwise).
func (p *Profile) addControl(c core.Control) {
	if p.byID == nil {
		p.byID = map[string]core.Control{}
	}
	if _, exists := p.byID[c.ID]; exists {
		return
	}
	p.Controls = append(p.Controls, c)
	p.byID[c.ID] = c
}

// ControlByID returns the control (for kind lookup during decode).
func (p *Profile) ControlByID(id string) (core.Control, bool) {
	c, ok := p.byID[id]
	return c, ok
}

func (p *Profile) addFader(cc uint8, id, label string) {
	c := core.Control{ID: id, Kind: core.ControlFader, Label: label}
	p.addControl(c)
	p.CCs[cc] = id
}
