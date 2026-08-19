package config

import (
	"bytes"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// ---------------------------------------------------------------------------
// Per-section export/import: mappings (bindings), sources, targets and
// profiles can be saved/moved individually (same YAML style as the full
// config). Payload format:
//
//   kind: bindings
//   bindings:
//     - source: wing
//       ...
// ---------------------------------------------------------------------------

const (
	SectionBindings = "bindings"
	SectionSources  = "sources"
	SectionTargets  = "targets"
	SectionProfiles = "profiles"
)

// Sections lists the individually savable config sections.
var Sections = []string{SectionBindings, SectionProfiles, SectionSources, SectionTargets}

// SectionFile is the transport format for one section.
type SectionFile struct {
	Kind     string          `yaml:"kind" json:"kind"`
	Bindings []Binding       `yaml:"bindings,omitempty" json:"bindings,omitempty"`
	Sources  []SourceConfig  `yaml:"sources,omitempty" json:"sources,omitempty"`
	Targets  []TargetConfig  `yaml:"targets,omitempty" json:"targets,omitempty"`
	Profiles []ProfileConfig `yaml:"profiles,omitempty" json:"profiles,omitempty"`
}

const (
	MergeUpsert  = "upsert"  // replace entries with matching keys, add new ones, keep the rest
	MergeReplace = "replace" // replace the whole section
)

// ParseSectionFile parses a YAML (or JSON — YAML is a superset) section file.
func ParseSectionFile(data []byte) (*SectionFile, error) {
	var sf SectionFile
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&sf); err != nil {
		return nil, err
	}
	sf.Kind = strings.ToLower(strings.TrimSpace(sf.Kind))
	if !contains(Sections, sf.Kind) {
		return nil, fmt.Errorf("kind must be one of %s", strings.Join(Sections, ", "))
	}
	if sf.Len() == 0 {
		return nil, fmt.Errorf("section file contains no entries")
	}
	return &sf, nil
}

// Len is the total number of entries across all section fields.
func (sf *SectionFile) Len() int {
	return len(sf.Bindings) + len(sf.Sources) + len(sf.Targets) + len(sf.Profiles)
}

// SectionData extracts one section from cfg for export.
func SectionData(cfg Config, kind string) (*SectionFile, error) {
	sf := &SectionFile{Kind: kind}
	switch kind {
	case SectionBindings:
		sf.Bindings = cfg.Bindings
	case SectionSources:
		sf.Sources = cfg.Sources
	case SectionTargets:
		sf.Targets = cfg.Targets
	case SectionProfiles:
		sf.Profiles = cfg.Profiles
	default:
		return nil, fmt.Errorf("unknown section %q", kind)
	}
	return sf, nil
}

// MergeSection merges the parsed section file into the config per mode
// ("upsert" by identity key, or "replace" wholesale). Call Normalize+Save+Reload
// afterwards (server.applyNewConfig does).
func (c *Config) MergeSection(sf *SectionFile, mode string) error {
	if mode != MergeUpsert && mode != MergeReplace {
		return fmt.Errorf("mode must be %q or %q", MergeUpsert, MergeReplace)
	}
	switch sf.Kind {
	case SectionBindings:
		c.Bindings = mergeByKey(c.Bindings, sf.Bindings, mode, func(b Binding) string { return b.Key() })
	case SectionSources:
		c.Sources = mergeByKey(c.Sources, sf.Sources, mode, func(s SourceConfig) string { return s.ID })
	case SectionTargets:
		c.Targets = mergeByKey(c.Targets, sf.Targets, mode, func(t TargetConfig) string { return t.ID })
	case SectionProfiles:
		c.Profiles = mergeByKey(c.Profiles, sf.Profiles, mode, func(p ProfileConfig) string { return p.ID })
	default:
		return fmt.Errorf("unknown section %q", sf.Kind)
	}
	return nil
}

// mergeByKey upserts (or wholesale replaces) slice items by identity key,
// preserving order: updated entries stay at their position, new ones append.
func mergeByKey[T any](existing, incoming []T, mode string, key func(T) string) []T {
	if mode == MergeReplace {
		return incoming
	}
	out := append([]T(nil), existing...)
	pos := make(map[string]int, len(out))
	for i, e := range out {
		pos[key(e)] = i
	}
	for _, in := range incoming {
		k := key(in)
		if i, ok := pos[k]; ok {
			out[i] = in
		} else {
			pos[k] = len(out)
			out = append(out, in)
		}
	}
	return out
}

// MarshalSectionYAML renders a section file in the canonical style.
func MarshalSectionYAML(sf *SectionFile) ([]byte, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(sf); err != nil {
		return nil, err
	}
	return buf.Bytes(), enc.Close()
}
