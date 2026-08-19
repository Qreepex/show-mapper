// Package core defines the domain model shared by all connectors:
// sources, targets, events, actions and the type registries.
//
// Nothing in here may import a concrete connector package
// (internal/sources/...  internal/targets/...) — connectors register
// themselves via the Registry and are looked up by name at runtime.
package core

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// ---------------------------------------------------------------------------
// Events
// ---------------------------------------------------------------------------

// EventKind classifies incoming source events.
type EventKind string

const (
	EventPressed  EventKind = "pressed"
	EventReleased EventKind = "released"
	EventValue    EventKind = "value" // analog value change (fader, encoder)
)

// Event is emitted by a Source when a physical control changes.
type Event struct {
	SourceID string    `json:"source"`
	Control  string    `json:"control"` // control ID within the source profile, e.g. "pad-0-3"
	Kind     EventKind `json:"kind"`
	Value    float64   `json:"value"` // normalized 0..1 (for buttons: 1 on press / 0 on release)
	Raw      uint8     `json:"raw"`   // raw 7-bit value (e.g. MIDI velocity / CC value)
	When     time.Time `json:"when"`
}

// ---------------------------------------------------------------------------
// Actions
// ---------------------------------------------------------------------------

// ActionKind matches config.ActionConfig.Type.
type ActionKind string

const (
	ActionKindCommand ActionKind = "command"
	ActionKindValue   ActionKind = "value"
	ActionKindFader   ActionKind = "fader"
)

// Action is the resolved, ready-to-send unit of work passed to a Target.
// Targets only ever see Actions — all binding/trigger logic lives in the
// Conductor, keeping target implementations dumb and small.
type Action struct {
	BindingID string     `json:"binding"`
	TargetID  string     `json:"target"`
	Kind      ActionKind `json:"kind"`
	Address   string     `json:"address"` // e.g. OSC address "/Page1/Fader201" / "/cmd"
	// Args holds the resolved payload:
	//   command: []any{string}            — e.g. "Go Executor 1.201"  (sent as OSC string)
	//   value:   []any{int32|float32}
	//   fader:   []any{int32|float32}     — source value scaled into the configured range
	Args []any `json:"args"`
}

// ---------------------------------------------------------------------------
// Connector interfaces
// ---------------------------------------------------------------------------

// Status is the runtime health of a connector instance.
type Status struct {
	State  string `json:"state"`  // "connected" | "connecting" | "disconnected" | "error"
	Detail string `json:"detail"` // human-readable detail, e.g. opened port or last error
}

const (
	StateConnected    = "connected"
	StateConnecting   = "connecting"
	StateDisconnected = "disconnected"
	StateError        = "error"
)

// Source is a connector that produces Events (MIDI board, timecode input, ...).
type Source interface {
	// Connect opens the underlying device/stream. Called after construction.
	// Implementations must be resilient: a missing device should put the
	// instance into StateError, not crash the process.
	Connect(ctx context.Context) error
	// Close releases all resources. Must be idempotent.
	Close() error
	// Events returns the channel the source emits on. Owned by the source;
	// drained by the Conductor.
	Events() <-chan Event
	// Status returns the last known status.
	Status() Status

	ID() string
	Type() string // registry type, e.g. "midi"
	// Controls describes the addressable controls (from the device profile),
	// used by the UI to offer dropdowns instead of free text.
	Controls() []Control
}

// Target is a connector that consumes Actions (OSC, future ArtNet/sACN, ...).
type Target interface {
	Connect(ctx context.Context) error
	Send(a Action) error
	Close() error
	Status() Status

	ID() string
	Type() string
}

// EventInjector is an optional interface for VIRTUAL sources (no hardware):
// it accepts events programmatically, e.g. the in-browser virtual surface
// (source type "sim") fed by the web UI via WS.
type EventInjector interface {
	Inject(ev Event) error
}

// FeedbackSink is an optional interface for sources that can update the
// visuals of their physical controls: LEDs on MIDI pads/buttons today,
// LCD key images/labels on Elgato Stream Decks next, etc.
// The Conductor uses it to reflect toggle state (and later, target feedback).
type FeedbackSink interface {
	SetControlFeedback(controlID string, fb ControlFeedback) error
}

// ControlFeedback is a visual-state request for one control.
// An implementation uses the subset its hardware supports and ignores the rest:
//   - LED-only sources (APC family) use LED and ignore Text/Icon.
//   - Display-based sources (Stream Deck key LCDs) may use Text/Icon (v0.2+).
type ControlFeedback struct {
	// LED requests a simple color/brightness/blink state.
	// nil means "leave the LED as is".
	LED *LEDState
	// Text is an overlay label for display-based keys (reserved).
	Text string
	// Icon is a rendered image for display-based keys (reserved).
	// nil means "leave the current image as is".
	Icon []byte
}

// LEDState is the abstract LED request; source profiles translate it into
// device-specific messages (e.g. RGB palette + blink-channel on APC mini mk2).
type LEDState struct {
	Mode  string `json:"mode"`  // "off" | "on" | "blink" | "pulse"
	Color string `json:"color"` // palette name, e.g. "red", "green", "amber" (profile-defined)
}

// ---------------------------------------------------------------------------
// Controls
// ---------------------------------------------------------------------------

// ControlKind describes the physical form of a control.
type ControlKind string

const (
	ControlPad     ControlKind = "pad"
	ControlButton  ControlKind = "button"
	ControlFader   ControlKind = "fader"
	ControlEncoder ControlKind = "encoder"
)

// Control describes one addressable control of a source device.
type Control struct {
	ID     string      `json:"id"`    // stable ID used in bindings, e.g. "pad-0-3", "fader-1"
	Label  string      `json:"label"` // human label, e.g. "Pad A1" (UI display)
	Kind   ControlKind `json:"kind"`
	Row    int         `json:"row,omitempty"` // grid position for pad rendering (0 = bottom row)
	Col    int         `json:"col,omitempty"`
	HasLED bool        `json:"hasLED"`
}

// ---------------------------------------------------------------------------
// Connector metadata (for the UI, see /api/meta)
// ---------------------------------------------------------------------------

// FieldSpec describes one connector option field so the web UI can render
// forms without hardcoding connector knowledge.
type FieldSpec struct {
	Name     string `json:"name"`
	Label    string `json:"label"`
	Type     string `json:"type"` // "text" | "number"
	Required bool   `json:"required"`
	Default  any    `json:"default,omitempty"`
	Help     string `json:"help,omitempty"`
}

// ProfileSummary describes a device profile (control map) of a source type,
// e.g. MIDI device layouts. Exposed to the UI so binding editors can offer
// real control dropdowns.
type ProfileSummary struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	LED      string    `json:"led"` // feedback style name or "none"
	Controls []Control `json:"controls"`
}

// TypeInfo describes a registered connector type.
type TypeInfo struct {
	Type     string           `json:"type"`               // registry key, e.g. "midi"
	Name     string           `json:"name"`               // human name, e.g. "USB-MIDI controller"
	Options  []FieldSpec      `json:"options"`            // option schema for the UI
	Profiles []ProfileSummary `json:"profiles,omitempty"` // source types only: device profiles
}

// Inspector is an optional connector capability (usually source types):
// enumerates hardware/devices the connector can see right now, e.g. MIDI
// ports or attached Stream Decks. Powers "detect device" pickers in the UI.
type Inspector func() (any, error)

// ---------------------------------------------------------------------------
// Errors
// ---------------------------------------------------------------------------

// permanentError marks startup failures where retrying is pointless
// (e.g. the binary was built without CGO and cannot speak MIDI at all).
// The conductor reports the error once instead of retrying in a loop.
type permanentError struct{ msg string }

func (e permanentError) Error() string { return e.msg }

// PermanentError wraps msg as a permanent (do-not-retry) error.
func PermanentError(msg string) error { return permanentError{msg: msg} }

// IsPermanent reports whether err is permanent.
func IsPermanent(err error) bool {
	var p permanentError
	return errors.As(err, &p)
}

// ---------------------------------------------------------------------------
// Sink (broadcast of backend activity to the web UI)
// ---------------------------------------------------------------------------

// Sink receives everything noteworthy; the server package forwards it to
// connected WebSocket clients. Kept as a tiny interface so core has no
// dependency on the HTTP layer.
type Sink interface {
	Broadcast(msgType string, data any)
}

// NopSink is used in tests and headless subcommands.
type NopSink struct{}

func (NopSink) Broadcast(string, any) {}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// ScaleFader maps a normalized value v (0..1) into [min,max] and clamps it.
func ScaleFader(v, min, max float64) float64 {
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	return min + v*(max-min)
}

func (s Status) String() string {
	if s.Detail == "" {
		return s.State
	}
	return fmt.Sprintf("%s (%s)", s.State, s.Detail)
}
