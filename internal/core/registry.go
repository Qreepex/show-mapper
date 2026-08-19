package core

import (
	"fmt"
	"sort"
	"sync"

	"github.com/Qreepex/show-mapper/internal/config"
)

// ---------------------------------------------------------------------------
// Source registry
// ---------------------------------------------------------------------------

// SourceBuildContext carries connector-relevant sections of the full config
// to a SourceFactory (e.g. user-defined device profiles).
type SourceBuildContext struct {
	// Profiles are the user-defined profiles from the config file `profiles:`
	// section; sources pick entries whose Type matches their connector type.
	Profiles []config.ProfileConfig
}

// SourceFactory builds a source instance from its config section.
// The factory must not open any device — that happens in Connect().
type SourceFactory func(cfg config.SourceConfig, bctx SourceBuildContext) (Source, error)

var (
	srcMu         sync.RWMutex
	srcFactories  = map[string]SourceFactory{}
	srcInfos      = map[string]TypeInfo{}
	srcInspectors = map[string]Inspector{}
)

// RegisterSource is called from init() of connector packages:
//
//	func init() { core.RegisterSource("midi", core.TypeInfo{...}, NewSource) }
//
// Registering the same type twice panics (programmer error, fail fast).
func RegisterSource(typ string, info TypeInfo, f SourceFactory) {
	srcMu.Lock()
	defer srcMu.Unlock()
	if _, dup := srcFactories[typ]; dup {
		panic("core: duplicate source type " + typ)
	}
	info.Type = typ
	srcFactories[typ] = f
	srcInfos[typ] = info
}

// RegisterInspector attaches an Inspector to a source type. Optional.
// Panics on duplicate registration for the same type.
func RegisterInspector(typ string, fn Inspector) {
	srcMu.Lock()
	defer srcMu.Unlock()
	if _, dup := srcInspectors[typ]; dup {
		panic("core: duplicate inspector for source type " + typ)
	}
	srcInspectors[typ] = fn
}

// InspectSourceType runs the inspector of a source type, if it has one.
// Returns (nil, nil) for types without an inspector.
func InspectSourceType(typ string) (any, error) {
	srcMu.RLock()
	fn, ok := srcInspectors[typ]
	srcMu.RUnlock()
	if !ok {
		return nil, nil
	}
	return fn()
}

// NewSource instantiates a source by config.
func NewSource(cfg config.SourceConfig, bctx SourceBuildContext) (Source, error) {
	srcMu.RLock()
	f, ok := srcFactories[cfg.Type]
	srcMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown source type %q (registered: %v)", cfg.Type, SourceTypes())
	}
	return f(cfg, bctx)
}

// SourceTypes returns registered source type names, sorted.
func SourceTypes() []string {
	srcMu.RLock()
	defer srcMu.RUnlock()
	out := make([]string, 0, len(srcFactories))
	for t := range srcFactories {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

// SourceTypeInfos returns metadata of all registered source types (for the UI).
func SourceTypeInfos() []TypeInfo {
	srcMu.RLock()
	defer srcMu.RUnlock()
	out := make([]TypeInfo, 0, len(srcInfos))
	for _, i := range srcInfos {
		out = append(out, i)
	}
	sort.Slice(out, func(a, b int) bool { return out[a].Type < out[b].Type })
	return out
}

// ---------------------------------------------------------------------------
// Target registry
// ---------------------------------------------------------------------------

// TargetFactory builds a target instance from its config section.
type TargetFactory func(cfg config.TargetConfig) (Target, error)

var (
	tgtMu        sync.RWMutex
	tgtFactories = map[string]TargetFactory{}
	tgtInfos     = map[string]TypeInfo{}
)

// RegisterTarget is called from init() of target packages.
func RegisterTarget(typ string, info TypeInfo, f TargetFactory) {
	tgtMu.Lock()
	defer tgtMu.Unlock()
	if _, dup := tgtFactories[typ]; dup {
		panic("core: duplicate target type " + typ)
	}
	info.Type = typ
	tgtFactories[typ] = f
	tgtInfos[typ] = info
}

// NewTarget instantiates a target by config.
func NewTarget(cfg config.TargetConfig) (Target, error) {
	tgtMu.RLock()
	f, ok := tgtFactories[cfg.Type]
	tgtMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown target type %q (registered: %v)", cfg.Type, TargetTypes())
	}
	return f(cfg)
}

// TargetTypes returns registered target type names, sorted.
func TargetTypes() []string {
	tgtMu.RLock()
	defer tgtMu.RUnlock()
	out := make([]string, 0, len(tgtFactories))
	for t := range tgtFactories {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

// TargetTypeInfos returns metadata of all registered target types.
func TargetTypeInfos() []TypeInfo {
	tgtMu.RLock()
	defer tgtMu.RUnlock()
	out := make([]TypeInfo, 0, len(tgtInfos))
	for _, i := range tgtInfos {
		out = append(out, i)
	}
	sort.Slice(out, func(a, b int) bool { return out[a].Type < out[b].Type })
	return out
}
