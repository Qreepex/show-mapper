// Package sim provides a virtual control surface source: no hardware, no
// CGO, no drivers. It exists so you can develop and demo the whole event →
// binding → action pipeline (and even run a show as a backup surface) from
// the browser: the UI page /surface renders this source's controls and
// injects events via WS (message type "client.sim", see docs/protocols.md).
//
// Control IDs intentionally mirror the APC mini mk2 profile (pad-r-c,
// button-track-N, fader-N, fader-master) so example bindings work 1:1.
package sim

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Qreepex/show-mapper/internal/config"
	"github.com/Qreepex/show-mapper/internal/core"
)

// ProfileID of the built-in virtual board.
const ProfileID = "sim-board"

func init() {
	core.RegisterSource("sim", core.TypeInfo{
		Name:     "Virtual surface (in-browser board; no hardware/CGO needed)",
		Profiles: []core.ProfileSummary{profileSummary()},
	}, NewSource)
}

// Source is a virtual control surface instance.
type Source struct {
	id     string
	events chan core.Event

	mu sync.Mutex
	st core.Status
}

// NewSource builds a sim instance (no options).
func NewSource(cfg config.SourceConfig, _ core.SourceBuildContext) (core.Source, error) {
	return &Source{
		id:     cfg.ID,
		events: make(chan core.Event, 128),
		st:     core.Status{State: core.StateDisconnected},
	}, nil
}

func (s *Source) ID() string                { return s.id }
func (s *Source) Type() string              { return "sim" }
func (s *Source) Events() <-chan core.Event { return s.events }
func (s *Source) Controls() []core.Control  { return controls }

// Connect is trivial for a virtual device.
func (s *Source) Connect(_ context.Context) error {
	s.setStatus(core.Status{State: core.StateConnected, Detail: "virtual — open /surface in the web UI"})
	return nil
}

func (s *Source) Close() error {
	s.setStatus(core.Status{State: core.StateDisconnected})
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

// Inject implements core.EventInjector — the entry point used by the web UI.
func (s *Source) Inject(ev core.Event) error {
	if !validControl(ev.Control) {
		return fmt.Errorf("unknown sim control %q", ev.Control)
	}
	switch ev.Kind {
	case core.EventPressed, core.EventReleased, core.EventValue:
	default:
		return fmt.Errorf("unsupported sim event kind %q", ev.Kind)
	}
	ev.SourceID = s.id
	ev.When = time.Now()
	select {
	case s.events <- ev:
	default:
	}
	return nil
}

// ---------------------------------------------------------------------------
// Built-in virtual board: same control scheme as apc-mini-mk2.
// ---------------------------------------------------------------------------

var controls = buildControls()
var controlSet = func() map[string]bool {
	m := map[string]bool{}
	for _, c := range controls {
		m[c.ID] = true
	}
	return m
}()

func validControl(id string) bool { return controlSet[id] }

func buildControls() []core.Control {
	out := []core.Control{}
	for n := 0; n < 64; n++ {
		row, col := n/8, n%8
		out = append(out, core.Control{
			ID: fmt.Sprintf("pad-%d-%d", row, col), Label: fmt.Sprintf("Pad %d/%d", row+1, col+1),
			Kind: core.ControlPad, Row: row, Col: col, HasLED: false,
		})
	}
	for i := 1; i <= 8; i++ {
		out = append(out, core.Control{
			ID: fmt.Sprintf("button-track-%d", i), Label: fmt.Sprintf("Button %d", i), Kind: core.ControlButton,
		})
		out = append(out, core.Control{
			ID: fmt.Sprintf("fader-%d", i), Label: fmt.Sprintf("Fader %d", i), Kind: core.ControlFader,
		})
	}
	out = append(out, core.Control{ID: "fader-master", Label: "Master", Kind: core.ControlFader})
	return out
}

func profileSummary() core.ProfileSummary {
	return core.ProfileSummary{
		ID: ProfileID, Name: "Virtual board (APC-style 8×8 + 8 buttons + 9 faders)",
		LED: "none", Controls: controls,
	}
}
