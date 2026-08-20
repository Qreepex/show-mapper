package gma3

import (
	"strings"
	"testing"

	"github.com/Qreepex/show-mapper/internal/core"
)

// This test binary's own init() would already panic on duplicate preset ids —
// keep these assertions as a registry sanity + resolution smoke test.
func TestPresetsRegisteredAndResolve(t *testing.T) {
	infos := core.ActionPresetInfos()
	if len(infos) < 10 {
		t.Fatalf("expected ≥10 gma3 presets, got %d", len(infos))
	}
	for _, p := range infos {
		if !strings.HasPrefix(p.ID, "gma3.") {
			t.Errorf("unexpected preset id %q", p.ID)
		}
		if len(p.TargetTypes) != 1 || p.TargetTypes[0] != "gma3" {
			t.Errorf("preset %q not scoped to gma3 target", p.ID)
		}
	}

	a, err := core.ResolveActionPreset("gma3.go", map[string]any{"page": 1, "executor": 201})
	if err != nil {
		t.Fatalf("resolve go: %v", err)
	}
	if a.Type != "command" || a.Address != "/cmd" || !strings.Contains(a.Command, "Page 1.201") {
		t.Errorf("bad resolved go action: %+v", a)
	}

	a, err = core.ResolveActionPreset("gma3.key", map[string]any{"page": "1", "executor": "201"})
	if err != nil {
		t.Fatalf("resolve key: %v", err)
	}
	if a.Type != "value" || a.PressValue == nil || a.ReleaseValue == nil {
		t.Errorf("bad resolved key action: %+v", a)
	}

	a, err = core.ResolveActionPreset("gma3.fader", map[string]any{"page": 1, "executor": 201})
	if err != nil {
		t.Fatalf("resolve fader: %v", err)
	}
	if a.Type != "fader" || a.Range == nil || a.Range[0] != 0 || a.Range[1] != 100 {
		t.Errorf("bad resolved fader: %+v", a)
	}

	if _, err := core.ResolveActionPreset("gma3.go", map[string]any{}); err == nil {
		t.Error("expected error for missing params")
	}
}
