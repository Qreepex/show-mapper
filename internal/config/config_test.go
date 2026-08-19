package config

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseDefaults(t *testing.T) {
	cfg, err := Parse([]byte(minimalYAML))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if cfg.HTTP.Listen != "127.0.0.1:8080" {
		t.Errorf("default listen not applied: %q", cfg.HTTP.Listen)
	}
	if got := cfg.Bindings[0].Mode; got != ModeMomentary {
		t.Errorf("default mode = %q, want momentary", got)
	}
	if got := cfg.Bindings[0].Action.ValueType; got != ValueTypeInt {
		t.Errorf("default valueType = %q, want int", got)
	}
}

func TestValidateDuplicateIDs(t *testing.T) {
	bad := strings.Replace(minimalYAML,
		"sources:\n  - id: wing\n    type: midi",
		"sources:\n  - id: wing\n    type: midi\n  - id: wing\n    type: midi",
		1)
	if bad == minimalYAML {
		t.Fatal("test setup failed: replacement did not apply")
	}
	_, err := Parse([]byte(bad))
	if err == nil || !strings.Contains(err.Error(), "duplicate source id") {
		t.Fatalf("expected duplicate source id error, got %v", err)
	}
}

func TestValidateBindingRules(t *testing.T) {
	bad := strings.Replace(minimalYAML, "address: /cmd", "address: cmd", 1)
	if _, err := Parse([]byte(bad)); err == nil {
		t.Fatal("expected error for address not starting with /")
	}

	badTrigger := strings.Replace(minimalYAML, "trigger: pressed", "trigger: yeeted", 1)
	if _, err := Parse([]byte(badTrigger)); err == nil {
		t.Fatal("expected error for invalid trigger")
	}

	missingCmd := strings.Replace(minimalYAML, "command: Go Executor 1.201", "", 1)
	if _, err := Parse([]byte(missingCmd)); err == nil {
		t.Fatal("expected error for command action without command")
	}
}

func TestValidateCustomProfile(t *testing.T) {
	good := minimalYAML + profileYAML
	if _, err := Parse([]byte(good)); err != nil {
		t.Fatalf("custom profile rejected: %v", err)
	}

	bad := strings.Replace(good, "note: 36", "note: 200", 1)
	if _, err := Parse([]byte(bad)); err == nil {
		t.Fatal("expected MIDI range error for note 200")
	}

	badBoth := strings.Replace(good, "note: 36", "note: 36\n        cc: 5", 1)
	if _, err := Parse([]byte(badBoth)); err == nil {
		t.Fatal("expected error when both note+cc set")
	}
}

func TestAutoCreate(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "sub", "show-mapper.yaml")
	f, err := CreateDefault(p)
	if err != nil {
		t.Fatalf("CreateDefault: %v", err)
	}
	if f.Path != p || f.Get().HTTP.Listen == "" {
		t.Fatalf("bad File: %+v", f)
	}
	back, err := LoadPath(p)
	if err != nil {
		t.Fatalf("LoadPath(created): %v", err)
	}
	if back.Get().HTTP.Listen != "127.0.0.1:8080" {
		t.Errorf("defaults lost: %+v", back.Get().HTTP)
	}

	// Setenv drives the SHOWMAPPER_CONFIG branch of PreferredCreatePath
	t.Setenv("SHOWMAPPER_CONFIG", p)
	if got := PreferredCreatePath(); got != p {
		t.Errorf("PreferredCreatePath = %q, want %q", got, p)
	}
}

func TestLoadErrNotFound(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("SHOWMAPPER_CONFIG", filepath.Join(t.TempDir(), "nope.yaml"))
	_, err := Load()
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestIDRule(t *testing.T) {
	for _, ok := range []string{"ma3", "wing-left", "apc-mini-mk2", "a"} {
		if !idRule.MatchString(ok) {
			t.Errorf("id %q should be valid", ok)
		}
	}
	for _, bad := range []string{"MA3", "wing_left", "-x", "", "a b"} {
		if idRule.MatchString(bad) {
			t.Errorf("id %q should be invalid", bad)
		}
	}
}

const minimalYAML = `
version: 1
sources:
  - id: wing
    type: midi
    profile: apc-mini-mk2
targets:
  - id: ma3
    type: osc
    options:
      host: 192.168.1.100
bindings:
  - source: wing
    control: pad-0-0
    trigger: pressed
    target: ma3
    action:
      type: command
      address: /cmd
      command: Go Executor 1.201
`

const profileYAML = `
profiles:
  - id: my-board
    type: midi
    name: My Custom Board
    match: ["MYBOARD"]
    led: { style: onOff }
    controls:
      - { id: pad-1, kind: pad, label: "Pad 1", note: 36, hasLED: true }
      - { id: fader-1, kind: fader, label: "Fader 1", cc: 1 }
`
