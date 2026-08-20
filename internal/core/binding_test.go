package core

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Qreepex/show-mapper/internal/config"
)

func TestScaleFader(t *testing.T) {
	cases := []struct {
		v, min, max, want float64
	}{
		{0, 0, 100, 0},
		{1, 0, 100, 100},
		{0.5, 0, 100, 50},
		{1.5, 0, 100, 100}, // clamped
		{-1, 0, 100, 0},    // clamped
		{0.5, 0, 255, 127.5},
	}
	for _, c := range cases {
		if got := ScaleFader(c.v, c.min, c.max); got != c.want {
			t.Errorf("ScaleFader(%v,%v,%v) = %v, want %v", c.v, c.min, c.max, got, c.want)
		}
	}
}

func TestNumericArg(t *testing.T) {
	if got := numericArg(49.5, config.ValueTypeInt); got != int32(50) {
		t.Errorf("int rounding: got %v (%T)", got, got)
	}
	if got := numericArg(49.5, config.ValueTypeFloat); got != float32(49.5) {
		t.Errorf("float: got %v (%T)", got, got)
	}
}

func TestPressReleaseActions(t *testing.T) {
	cmd := config.Binding{
		Source: "s", Control: "pad-1", Trigger: config.TriggerPressed, Target: "t",
		Action: config.ActionConfig{Type: config.ActionCommand, Address: "/cmd",
			Command: "Go+ Page 1.201", ReleaseCommand: "Off Page 1.201"},
	}
	a, ok := pressAction(cmd)
	if !ok || a.Address != "/cmd" || a.Args[0] != "Go+ Page 1.201" {
		t.Errorf("pressAction: %+v ok=%v", a, ok)
	}
	a, ok = releaseAction(cmd)
	if !ok || a.Args[0] != "Off Page 1.201" {
		t.Errorf("releaseAction: %+v ok=%v", a, ok)
	}
}

func TestValueActions(t *testing.T) {
	press := 1.0
	val := config.Binding{
		Source: "s", Control: "pad-1", Trigger: config.TriggerPressed, Target: "t",
		Action: config.ActionConfig{Type: config.ActionValue, Address: "/x", PressValue: &press},
	}
	a, ok := pressAction(val)
	if !ok || a.Args[0] != int32(1) {
		t.Errorf("value pressAction: %+v ok=%v", a, ok)
	}
	a, ok = releaseAction(val)
	if !ok || a.Args[0] != int32(1) { // fallback: press value when ReleaseValue unset
		t.Errorf("value releaseAction fallback: %+v ok=%v", a, ok)
	}

	fader := config.Binding{
		Source: "s", Control: "fader-1", Trigger: config.TriggerValue, Target: "t",
		Action: config.ActionConfig{Type: config.ActionFader, Address: "/f", Range: &[2]float64{0, 200}},
	}
	a, ok = valueAction(fader, 0.25)
	if !ok || a.Args[0] != int32(50) {
		t.Errorf("valueAction: %+v ok=%v (want arg int32(50))", a, ok)
	}
}

// ---------------------------------------------------------------------------
// Conductor end-to-end with fake connectors
// ---------------------------------------------------------------------------

type fakeSource struct {
	cfg    config.SourceConfig
	events chan Event
}

func (f *fakeSource) Connect(context.Context) error { return nil }
func (f *fakeSource) Close() error                  { return nil }
func (f *fakeSource) Events() <-chan Event          { return f.events }
func (f *fakeSource) Status() Status                { return Status{State: StateConnected} }
func (f *fakeSource) ID() string                    { return f.cfg.ID }
func (f *fakeSource) Type() string                  { return "fake" }
func (f *fakeSource) Controls() []Control           { return nil }

type fakeTarget struct {
	cfg  config.TargetConfig
	sent chan Action
}

func (f *fakeTarget) Connect(context.Context) error { return nil }
func (f *fakeTarget) Close() error                  { return nil }
func (f *fakeTarget) Status() Status                { return Status{State: StateConnected} }
func (f *fakeTarget) ID() string                    { return f.cfg.ID }
func (f *fakeTarget) Type() string                  { return "fake" }
func (f *fakeTarget) Send(a Action) error           { f.sent <- a; return nil }

var (
	fakesMu   sync.Mutex
	fakesSrcs = map[string]*fakeSource{}
	fakesTgts = map[string]*fakeTarget{}
)

func resetFakes() {
	fakesMu.Lock()
	fakesSrcs = map[string]*fakeSource{}
	fakesTgts = map[string]*fakeTarget{}
	fakesMu.Unlock()
}

func fakeSrc(id string) *fakeSource {
	fakesMu.Lock()
	defer fakesMu.Unlock()
	return fakesSrcs[id]
}

func fakeTgt(id string) *fakeTarget {
	fakesMu.Lock()
	defer fakesMu.Unlock()
	return fakesTgts[id]
}

func init() {
	RegisterSource("fake", TypeInfo{Name: "fake source"}, func(cfg config.SourceConfig, _ SourceBuildContext) (Source, error) {
		s := &fakeSource{cfg: cfg, events: make(chan Event, 16)}
		fakesMu.Lock()
		fakesSrcs[cfg.ID] = s
		fakesMu.Unlock()
		return s, nil
	})
	RegisterTarget("fake", TypeInfo{Name: "fake target"}, func(cfg config.TargetConfig) (Target, error) {
		tg := &fakeTarget{cfg: cfg, sent: make(chan Action, 16)}
		fakesMu.Lock()
		fakesTgts[cfg.ID] = tg
		fakesMu.Unlock()
		return tg, nil
	})
}

func waitFor[T any](get func() (T, bool)) (T, error) {
	deadline := time.Now().Add(2 * time.Second)
	for {
		if v, ok := get(); ok {
			return v, nil
		}
		if time.Now().After(deadline) {
			var zero T
			return zero, fmt.Errorf("timeout")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestConductorDispatchMomentary(t *testing.T) {
	resetFakes()
	pressV, relV := 1.0, 0.0
	cfg := config.Config{
		Version: 1,
		HTTP:    config.HTTPConfig{Listen: "127.0.0.1:0"},
		Sources: []config.SourceConfig{{ID: "wing", Type: "fake"}},
		Targets: []config.TargetConfig{{ID: "out", Type: "fake"}},
		Bindings: []config.Binding{{
			Source: "wing", Control: "pad-0-0", Trigger: config.TriggerPressed, Target: "out",
			Action: config.ActionConfig{Type: config.ActionValue, Address: "/Page1/Key201",
				PressValue: &pressV, ReleaseValue: &relV},
		}},
	}

	c := NewConductor(cfg, NopSink{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go c.Run(ctx)

	src, err := waitFor(func() (*fakeSource, bool) { s := fakeSrc("wing"); return s, s != nil })
	if err != nil {
		t.Fatal("source not started:", err)
	}
	src.events <- Event{SourceID: "wing", Control: "pad-0-0", Kind: EventPressed, Value: 1, When: time.Now()}

	act, err := waitFor(func() (Action, bool) {
		tgt := fakeTgt("out")
		if tgt == nil {
			return Action{}, false
		}
		select {
		case a := <-tgt.sent:
			return a, true
		default:
			return Action{}, false
		}
	})
	if err != nil {
		t.Fatal("no action dispatched:", err)
	}
	if act.Address != "/Page1/Key201" || act.TargetID != "out" {
		t.Errorf("unexpected action: %+v", act)
	}
	if act.Args[0] != int32(1) {
		t.Errorf("press arg = %v, want int32(1)", act.Args[0])
	}
}
