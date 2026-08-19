package midi

import (
	"fmt"
	"strings"

	"github.com/yourorg/showbridge/internal/config"
	"github.com/yourorg/showbridge/internal/core"
)

// ---------------------------------------------------------------------------
// User-defined ("custom") MIDI device profiles
//
// Built-in profiles ship with the binary (profile_apc.go). Users can describe
// ANY other board in the config file's `profiles:` section (or via the UI's
// Boards editor / MIDI learn workflow) — see docs/midi-devices.md#custom-boards.
//
// resolveProfile picks the profile for a source instance:
//
//	1. explicit `profile:` id  → built-ins first, then customs
//	2. `device:` port name     → built-in match, then custom match
//	3. error with guidance
// ---------------------------------------------------------------------------

func resolveProfile(cfg config.SourceConfig, customs []config.ProfileConfig) (*Profile, error) {
	if cfg.Profile != "" {
		if p, ok := ProfileByID(cfg.Profile); ok {
			return p, nil
		}
		for _, pc := range customs {
			if pc.Type == "midi" && pc.ID == cfg.Profile {
				return ProfileFromConfig(pc)
			}
		}
		return nil, fmt.Errorf("source %q: unknown MIDI profile %q (built-in: %v — or define a custom one under `profiles:`)",
			cfg.ID, cfg.Profile, ProfileIDs())
	}

	device := config.OptionString(cfg.Options, "device", "")
	if device == "" {
		return nil, fmt.Errorf("source %q: set `profile` (%v, or a custom id) or a `device` name to match", cfg.ID, ProfileIDs())
	}
	if p := ProfileForDevice(device); p != nil {
		return p, nil
	}
	lower := strings.ToLower(device)
	for _, pc := range customs {
		if pc.Type != "midi" {
			continue
		}
		p, err := ProfileFromConfig(pc)
		if err != nil {
			return nil, fmt.Errorf("source %q: custom profile %q is invalid: %w", cfg.ID, pc.ID, err)
		}
		for _, m := range pc.Match {
			if strings.Contains(lower, strings.ToLower(m)) {
				return p, nil
			}
		}
	}
	return nil, fmt.Errorf("source %q: no profile matches device %q — set `profile` explicitly, add `match` to a custom profile, or create one (see docs/midi-devices.md#custom-boards)",
		cfg.ID, device)
}

// ProfileFromConfig converts a user-defined profile (config file / UI) into a
// runtime Profile. The config package already validated structurally; this
// returns errors only for midi-level inconsistencies.
func ProfileFromConfig(pc config.ProfileConfig) (*Profile, error) {
	if pc.Type != "midi" {
		return nil, fmt.Errorf("profile %q: type %q is not \"midi\"", pc.ID, pc.Type)
	}
	if _, clash := ProfileByID(pc.ID); clash {
		return nil, fmt.Errorf("profile %q: id collides with a built-in profile", pc.ID)
	}

	led, err := ledFromSpec(pc.LED)
	if err != nil {
		return nil, fmt.Errorf("profile %q: %w", pc.ID, err)
	}

	p := newProfile(pc.ID, pc.Name, pc.Match, led, nil)
	for _, c := range pc.Controls {
		ctl := core.Control{
			ID:     c.ID,
			Kind:   core.ControlKind(c.Kind),
			Label:  c.Label,
			HasLED: c.HasLED,
		}
		if ctl.Label == "" {
			ctl.Label = c.ID
		}
		if c.Row != nil {
			ctl.Row = *c.Row
		}
		if c.Col != nil {
			ctl.Col = *c.Col
		}
		p.Controls = append(p.Controls, ctl)

		switch {
		case c.Note != nil:
			n := uint8(*c.Note)
			p.Notes[n] = c.ID
			if c.HasLED {
				ledNote := n
				if c.LEDNote != nil {
					ledNote = uint8(*c.LEDNote)
				}
				p.NoteOf[c.ID] = ledNote
			}
		case c.CC != nil:
			p.CCs[uint8(*c.CC)] = c.ID
		}
	}
	return p, nil
}

// ledFromSpec picks the LED backend for a custom profile.
func ledFromSpec(spec config.ProfileLED) (ledBackend, error) {
	switch spec.Style {
	case "", "none":
		return ledNone{}, nil
	case "onOff":
		on := spec.OnVelocity
		if on == 0 {
			on = 127
		}
		return ledOnOff{onVel: uint8(on)}, nil
	case "velocity":
		return ledVelocity{colors: colorTable(spec.Colors)}, nil
	case "apc2-rgb":
		return ledAPC2{}, nil
	}
	return nil, fmt.Errorf("unknown led.style %q", spec.Style)
}

func colorTable(custom map[string]int) map[string]uint8 {
	// Defaults follow the community-documented APC mini mk1 scheme.
	t := map[string]uint8{"green": 1, "red": 3, "yellow": 5, "amber": 5, "orange": 5}
	for name, v := range custom {
		if v >= 0 && v <= 127 {
			t[name] = uint8(v)
		}
	}
	return t
}

// ledNone is the fallback for boards/controls without LEDs.
type ledNone struct{}

func (ledNone) Name() string { return "none" }
func (ledNone) Messages(uint8, core.LEDState) [][]byte {
	return nil
}

// ledOnOff: on = NoteOn(onVel), off = NoteOff/vel 0.
// Works for most boards with simple single-color LEDs.
type ledOnOff struct{ onVel uint8 }

func (l ledOnOff) Name() string { return "onOff" }
func (l ledOnOff) Messages(note uint8, st core.LEDState) [][]byte {
	if st.Mode == "off" {
		return [][]byte{{0x90, note, 0}}
	}
	// blink/pulse have no hardware equivalent — stay on
	return [][]byte{{0x90, note, l.onVel}}
}

// ledVelocity: color name -> velocity (APC mk1-style scheme, table configurable).
type ledVelocity struct{ colors map[string]uint8 }

func (ledVelocity) Name() string { return "velocity-color" }
func (l ledVelocity) Messages(note uint8, st core.LEDState) [][]byte {
	if st.Mode == "off" {
		return [][]byte{{0x90, note, 0}}
	}
	vel, ok := l.colors[st.Color]
	if !ok {
		vel = 1 // green-ish default
	}
	return [][]byte{{0x90, note, vel}}
}
