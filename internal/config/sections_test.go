package config

import (
	"strings"
	"testing"
)

const sectionYAML = `
kind: bindings
bindings:
  - id: key-201
    source: wing
    control: pad-0-0
    trigger: pressed
    target: ma3
    action:
      type: value
      address: /Page1/Key201
      pressValue: 1
      releaseValue: 0
  - id: new-one
    source: wing
    control: pad-0-1
    trigger: pressed
    target: ma3
    action:
      type: value
      address: /Page1/Key202
      pressValue: 1
`

func sectionFixture(t *testing.T) Config {
	t.Helper()
	cfg, err := Parse([]byte(minimalYAML))
	if err != nil {
		t.Fatalf("fixture parse: %v", err)
	}
	cfg.Bindings[0].ID = "key-201"
	cfg.Bindings[0].Action.Type = ActionCommand
	return cfg
}

func TestParseSectionFile(t *testing.T) {
	sf, err := ParseSectionFile([]byte(sectionYAML))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if sf.Kind != SectionBindings || len(sf.Bindings) != 2 {
		t.Fatalf("got %+v", sf)
	}

	if _, err := ParseSectionFile([]byte("kind: nope\nbindings: []")); err == nil {
		t.Fatal("expected kind validation error")
	}
}

func TestMergeUpsert(t *testing.T) {
	cfg := sectionFixture(t)
	sf, _ := ParseSectionFile([]byte(sectionYAML))

	if err := cfg.MergeSection(sf, MergeUpsert); err != nil {
		t.Fatalf("merge: %v", err)
	}
	if len(cfg.Bindings) != 2 {
		t.Fatalf("upsert must yield 2 bindings (replace key-201, add new-one), got %d", len(cfg.Bindings))
	}
	if cfg.Bindings[0].ID != "key-201" || cfg.Bindings[0].Action.Type != ActionValue {
		t.Errorf("key-201 should be replaced with the value-typed import version")
	}
	if cfg.Bindings[1].ID != "new-one" {
		t.Errorf("new-one must be appended, got %q", cfg.Bindings[1].ID)
	}

	// replace mode wipes others
	sf.Bindings = sf.Bindings[:1]
	if err := cfg.MergeSection(sf, MergeReplace); err != nil {
		t.Fatalf("merge replace: %v", err)
	}
	if len(cfg.Bindings) != 1 || cfg.Bindings[0].ID != "key-201" {
		t.Errorf("replace should leave exactly key-201, got %+v", cfg.Bindings)
	}
}

func TestMergeTargetsByID(t *testing.T) {
	cfg := sectionFixture(t)
	sec := `
kind: targets
targets:
  - id: ma3
    type: osc
    options: { host: 10.0.0.99, port: 8000 }
  - id: ma3-backup
    type: osc
    options: { host: 10.0.0.100, port: 8000 }
`
	sf, err := ParseSectionFile([]byte(sec))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if err := cfg.MergeSection(sf, MergeUpsert); err != nil {
		t.Fatalf("merge: %v", err)
	}
	if got := cfg.Targets[0].Options["host"]; got != "10.0.0.99" {
		t.Errorf("existing target not updated: %v", got)
	}
	if len(cfg.Targets) != 2 || cfg.Targets[1].ID != "ma3-backup" {
		t.Errorf("new target not appended: %+v", cfg.Targets)
	}
	if err := cfg.MergeSection(sf, "banana"); err == nil {
		t.Fatal("bad mode must error")
	}
}

func TestSectionYAMLRoundTrip(t *testing.T) {
	cfg := sectionFixture(t)
	sf, err := SectionData(cfg, SectionBindings)
	if err != nil {
		t.Fatalf("SectionData: %v", err)
	}
	data, err := MarshalSectionYAML(sf)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	back, err := ParseSectionFile(data)
	if err != nil {
		t.Fatalf("roundtrip parse: %v", err)
	}
	if back.Kind != SectionBindings || len(back.Bindings) != len(cfg.Bindings) {
		t.Fatalf("roundtrip mismatch: %+v", back.Bindings)
	}
	// canonical style is 2-space indented (hand-editable with the example files)
	if strings.Contains(string(data), "\n    - id:") {
		t.Error("expected 2-space list indent")
	}
}
