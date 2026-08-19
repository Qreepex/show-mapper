package core

import (
	"context"
	"testing"
	"time"

	"github.com/Qreepex/show-mapper/internal/config"
)

// Registers a fake preset once for dispatch tests.
func init() {
	RegisterActionPreset(
		ActionPreset{ID: "fake.go", Source: "test", Label: "fake go",
			Fields: []FieldSpec{{Name: "page"}, {Name: "executor"}}},
		func(params map[string]any) (config.ActionConfig, error) {
			return config.ActionConfig{
				Type: config.ActionCommand, Address: "/cmd",
				Command: "Go Executor " + params["page"].(string) + "." + params["executor"].(string),
			}, nil
		},
	)
}

func TestConductorDispatchPresetBinding(t *testing.T) {
	resetFakes()
	cfg := config.Config{
		Version: 1,
		HTTP:    config.HTTPConfig{Listen: "127.0.0.1:0"},
		Sources: []config.SourceConfig{{ID: "wing", Type: "fake"}},
		Targets: []config.TargetConfig{{ID: "out", Type: "fake"}},
		Bindings: []config.Binding{{
			Source: "wing", Control: "pad-0-0", Trigger: config.TriggerPressed, Target: "out",
			Action: config.ActionConfig{Type: config.ActionTypePreset, Preset: "fake.go",
				Params: map[string]any{"page": "3", "executor": "104"}},
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
	if act.Address != "/cmd" || act.Args[0] != "Go Executor 3.104" {
		t.Errorf("preset resolution wrong: %+v", act)
	}
}

func TestPresetLoopGuardAndMissing(t *testing.T) {
	// unknown preset → resolve fails, press yields no action
	b := config.Binding{
		Source: "s", Control: "pad-1", Trigger: config.TriggerPressed, Target: "t",
		Action: config.ActionConfig{Type: config.ActionTypePreset, Preset: "does-not-exist"},
	}
	if _, ok := pressAction(b); ok {
		t.Error("pressAction should be !ok for unknown preset")
	}
}
