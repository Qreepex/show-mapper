package midi

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/Qreepex/show-mapper/internal/config"
	"github.com/Qreepex/show-mapper/internal/core"
)

func init() {
	core.RegisterSource("midi", core.TypeInfo{
		Name: "USB-MIDI controller (class-compliant; APC mini family, ...)",
		Options: []core.FieldSpec{
			{
				Name:  "device",
				Label: "Device name",
				Type:  "text",
				Help:  "Case-insensitive substring of the MIDI port name, e.g. \"APC mini mk2\". Leave empty to use the profile's default match. List available ports with `show-mapper midi list` or GET /api/sources/midi/inspect.",
			},
		},
		Profiles: Profiles(),
	}, NewSource)
	core.RegisterInspector("midi", InspectPorts)
}

// PortList is the result of InspectPorts.
type PortList struct {
	In  []PortInfo `json:"in"`
	Out []PortInfo `json:"out"`
}

// InspectPorts enumerates MIDI ports (used by GET /api/sources/midi/inspect
// and `show-mapper midi list`). Fails with ErrNoCGO in non-CGO builds.
func InspectPorts() (any, error) {
	hw, err := NewHW()
	if err != nil {
		return nil, err
	}
	in, err := hw.InPorts()
	if err != nil {
		return nil, err
	}
	out, err := hw.OutPorts()
	if err != nil {
		return nil, err
	}
	return PortList{In: in, Out: out}, nil
}

// Source is a connector instance bound to one physical MIDI device.
type Source struct {
	id      string
	device  string
	profile *Profile

	events chan core.Event

	mu     sync.Mutex
	st     core.Status
	conn   Conn
	closed bool
}

// NewSource builds a MIDI source from config. The profile is resolved from
// built-ins (apc-mini-mk2, apc-mini) or user-defined custom profiles
// (config `profiles:` section with type "midi").
func NewSource(cfg config.SourceConfig, bctx core.SourceBuildContext) (core.Source, error) {
	prof, err := resolveProfile(cfg, bctx.Profiles)
	if err != nil {
		return nil, err
	}
	device := config.OptionString(cfg.Options, "device", "")
	if device == "" && len(prof.Match) > 0 {
		device = prof.Match[0]
	}

	return &Source{
		id:      cfg.ID,
		device:  device,
		profile: prof,
		events:  make(chan core.Event, 128),
		st:      core.Status{State: core.StateDisconnected},
	}, nil
}

func (s *Source) ID() string                { return s.id }
func (s *Source) Type() string              { return "midi" }
func (s *Source) Events() <-chan core.Event { return s.events }
func (s *Source) Controls() []core.Control  { return s.profile.Controls }

// Connect opens input (and, if available, output) ports for the device.
func (s *Source) Connect(_ context.Context) error {
	s.setStatus(core.Status{State: core.StateConnecting, Detail: "opening " + s.device})

	hw, err := NewHW()
	if err != nil {
		// e.g. ErrNoCGO — retrying forever would never succeed.
		s.setStatus(core.Status{State: core.StateError, Detail: err.Error()})
		return core.PermanentError(err.Error())
	}

	conn, err := hw.Open(s.device, s.onRaw)
	if err != nil {
		// device unplugged / not present — retryable
		s.setStatus(core.Status{State: core.StateError, Detail: err.Error()})
		return err
	}

	s.mu.Lock()
	s.conn = conn
	s.closed = false
	s.mu.Unlock()

	// Device handshake (APC mini mk2 wants the "Introduction" sysex before
	// other device-specific traffic); failure is non-fatal.
	if inner := s.profile.IntroSysex; len(inner) > 0 {
		msg := make([]byte, 0, len(inner)+2)
		msg = append(msg, 0xF0)
		msg = append(msg, inner...)
		msg = append(msg, 0xF7)
		if err := conn.Send(msg); err != nil {
			slog.Debug("midi: sysex intro failed", "source", s.id, "err", err)
		}
	}

	detail := conn.InPortName()
	if out := conn.OutPortName(); out != "" {
		detail += " ⇄ " + out
	} else {
		detail += " (no output: LED feedback disabled)"
	}
	s.setStatus(core.Status{State: core.StateConnected, Detail: detail})
	return nil
}

func (s *Source) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	if s.conn != nil {
		_ = s.conn.Close()
		s.conn = nil
	}
	s.st = core.Status{State: core.StateDisconnected}
	return nil
}

func (s *Source) Status() core.Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.st
}

func (s *Source) setStatus(st core.Status) {
	s.mu.Lock()
	s.st = st
	s.mu.Unlock()
}

// onRaw decodes the channel-voice subset of MIDI (Note On/Off, CC) and turns
// it into core.Events. Sysex and realtime traffic is ignored for now.
// Unknown/unmapped controls are still published with synthetic control IDs
// ("note:NN" / "cc:NN") — this powers the UI's "learn" workflow.
func (s *Source) onRaw(data []byte) {
	if len(data) < 3 {
		return
	}
	// channel (data[0] & 0x0F) is intentionally ignored: boards like the mk2
	// use different channels per mode (drum mode = ch 9) — controls stay the same.
	var ev core.Event
	switch data[0] & 0xF0 {
	case 0x90: // Note On
		note, vel := data[1], data[2]
		if vel == 0 { // running-status quirks: NoteOn vel=0 == NoteOff
			ev = s.noteEvent(note, core.EventReleased, 0)
		} else {
			ev = s.noteEvent(note, core.EventPressed, vel)
		}
	case 0x80: // Note Off
		ev = s.noteEvent(data[1], core.EventReleased, 0)
	case 0xB0: // Control Change
		cc, val := data[1], data[2]
		ctrl, ok := s.profile.CCs[cc]
		if !ok {
			ctrl = fmt.Sprintf("cc:%d", cc)
		}
		ev = core.Event{Kind: core.EventValue, Control: ctrl, Value: float64(val) / 127.0, Raw: val}
	default:
		return
	}

	ev.SourceID = s.id
	ev.When = time.Now()

	s.mu.Lock()
	closed := s.closed
	s.mu.Unlock()
	if closed {
		return
	}
	select {
	case s.events <- ev:
	default: // consumer gone / flooded: drop rather than block the driver callback
	}
}

func (s *Source) noteEvent(note uint8, kind core.EventKind, rawVel uint8) core.Event {
	ctrl, ok := s.profile.Notes[note]
	if !ok {
		ctrl = fmt.Sprintf("note:%d", note)
	}
	v := 0.0
	if kind == core.EventPressed {
		v = 1.0
	}
	return core.Event{Kind: kind, Control: ctrl, Value: v, Raw: rawVel}
}

// SetControlFeedback implements core.FeedbackSink. MIDI boards only support
// the LED part of the request; Text/Icon are ignored.
func (s *Source) SetControlFeedback(controlID string, fb core.ControlFeedback) error {
	if fb.LED == nil {
		return nil
	}
	st := *fb.LED
	note, ok := s.profile.NoteOf[controlID]
	if !ok {
		return fmt.Errorf("control %q has no LED", controlID)
	}
	s.mu.Lock()
	conn := s.conn
	s.mu.Unlock()
	if conn == nil {
		return fmt.Errorf("device not connected")
	}
	for _, msg := range s.profile.LED.Messages(note, st) {
		if err := conn.Send(msg); err != nil {
			return err
		}
	}
	return nil
}
