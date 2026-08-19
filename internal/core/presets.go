package core

import (
	"fmt"
	"sort"
	"sync"

	"github.com/yourorg/showbridge/internal/config"
)

// ---------------------------------------------------------------------------
// Action presets: authoring-time helpers offered by helper modules.
//
// Connectors stay 100% generic (the osc target knows nothing about grandMA3).
// "Helper modules" (internal/helpers/<name>) instead register ready-made
// action builders — e.g. grandMA3 "Go/Flash/Temp/..." — which the UI offers as
// presets and resolves server-side into a plain config.ActionConfig that is
// stored in the binding. This keeps showbridge itself completely generic:
// presets are sugar, never required.
// ---------------------------------------------------------------------------

// ActionPreset describes one preset (and its parameter form) for the UI.
type ActionPreset struct {
	ID     string      `json:"id"`     // namespaced, e.g. "gma3.go"
	Source string      `json:"source"` // providing module, e.g. "gma3"
	Label  string      `json:"label"`
	Help   string      `json:"help,omitempty"`
	Fields []FieldSpec `json:"fields"` // parameter form (rendered by the UI)
}

type presetEntry struct {
	info    ActionPreset
	factory ActionPresetFactory
}

// ActionPresetFactory converts user-filled parameters to a concrete action.
type ActionPresetFactory func(params map[string]any) (config.ActionConfig, error)

var (
	presetMu sync.RWMutex
	presets  = map[string]presetEntry{}
)

// RegisterActionPreset is called from init() of helper modules.
// Duplicate IDs panic (programmer error).
func RegisterActionPreset(info ActionPreset, f ActionPresetFactory) {
	presetMu.Lock()
	defer presetMu.Unlock()
	if info.ID == "" {
		panic("core: action preset without id")
	}
	if _, dup := presets[info.ID]; dup {
		panic("core: duplicate action preset " + info.ID)
	}
	presets[info.ID] = presetEntry{info: info, factory: f}
}

// ResolveActionPreset builds an ActionConfig from preset parameters.
func ResolveActionPreset(id string, params map[string]any) (config.ActionConfig, error) {
	presetMu.RLock()
	e, ok := presets[id]
	presetMu.RUnlock()
	if !ok {
		return config.ActionConfig{}, fmt.Errorf("unknown action preset %q", id)
	}
	return e.factory(params)
}

// ActionPresetInfos lists all presets (sorted by ID) for /api/meta.
func ActionPresetInfos() []ActionPreset {
	presetMu.RLock()
	defer presetMu.RUnlock()
	out := make([]ActionPreset, 0, len(presets))
	for _, e := range presets {
		out = append(out, e.info)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
