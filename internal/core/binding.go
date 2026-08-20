package core

import (
	"fmt"
	"log/slog"
	"math"

	"github.com/Qreepex/show-mapper/internal/config"
)

// This file resolves config.Bindings into concrete Actions.
// It lives in core (not in config) because the Action type lives here and
// config may not import core (import cycle). All trigger/mode *matching*
// happens in conductor.go; these helpers only resolve payloads.

// PressAction resolves the payload for a "press" (or hold-fire, toggle-on).
// ok == false means "configured to do nothing".
func pressAction(b config.Binding) (Action, bool) {
	if b.Action.Type == config.ActionTypePreset {
		b2, err := resolvePreset(b)
		if err != nil {
			slog.Warn("preset resolve failed", "binding", b.Key(), "preset", b.Action.Preset, "err", err)
			return Action{}, false
		}
		return pressAction(b2)
	}
	base := Action{BindingID: b.Key(), TargetID: b.Target, Kind: ActionKind(b.Action.Type), Address: b.Action.Address}
	switch b.Action.Type {
	case config.ActionCommand:
		if b.Action.Command == "" {
			return Action{}, false
		}
		base.Args = []any{b.Action.Command}
		return base, true
	case config.ActionValue:
		if b.Action.PressValue == nil {
			return Action{}, false
		}
		base.Args = []any{numericArg(*b.Action.PressValue, b.Action.ValueType)}
		return base, true
	}
	// fader bindings react to "value" triggers only
	return Action{}, false
}

// ReleaseAction resolves the payload for a "release" (toggle-off / momentary
// pair side / released-trigger edge). Falls back to the press payload when no
// release-specific value is configured — so `trigger: released` bindings work
// with only a press payload set.
func releaseAction(b config.Binding) (Action, bool) {
	if b.Action.Type == config.ActionTypePreset {
		b2, err := resolvePreset(b)
		if err != nil {
			slog.Warn("preset resolve failed", "binding", b.Key(), "preset", b.Action.Preset, "err", err)
			return Action{}, false
		}
		return releaseAction(b2)
	}
	base := Action{BindingID: b.Key(), TargetID: b.Target, Kind: ActionKind(b.Action.Type), Address: b.Action.Address}
	switch b.Action.Type {
	case config.ActionCommand:
		if b.Action.ReleaseCommand != "" {
			base.Args = []any{b.Action.ReleaseCommand}
			return base, true
		}
		// release trigger without a release command: fall back to press command
		if b.Action.Command != "" {
			base.Args = []any{b.Action.Command}
			return base, true
		}
		return Action{}, false
	case config.ActionValue:
		v := b.Action.ReleaseValue
		if v == nil {
			v = b.Action.PressValue // fallback for release-trigger bindings
		}
		if v == nil {
			return Action{}, false
		}
		base.Args = []any{numericArg(*v, b.Action.ValueType)}
		return base, true
	}
	return Action{}, false
}

// ValueAction resolves the payload for an analog "value" trigger:
// the normalized source value (0..1) is scaled into the configured range.
func valueAction(b config.Binding, v float64) (Action, bool) {
	if b.Action.Type == config.ActionTypePreset {
		b2, err := resolvePreset(b)
		if err != nil {
			slog.Warn("preset resolve failed", "binding", b.Key(), "preset", b.Action.Preset, "err", err)
			return Action{}, false
		}
		return valueAction(b2, v)
	}
	if b.Action.Type != config.ActionFader || b.Action.Range == nil {
		return Action{}, false
	}
	scaled := ScaleFader(v, b.Action.Range[0], b.Action.Range[1])
	return Action{
		BindingID: b.Key(),
		TargetID:  b.Target,
		Kind:      ActionKindFader,
		Address:   b.Action.Address,
		Args:      []any{numericArg(scaled, b.Action.ValueType)},
	}, true
}

// numericArg converts a float to the wire type: int32 (rounded) or float32.
// grandMA3 expects ints for executor faders (see internal/helpers/gma3/README.md).
func numericArg(v float64, valueType string) any {
	if valueType == config.ValueTypeFloat {
		return float32(v)
	}
	return int32(int64(math.Round(v)))
}

// resolvePreset swaps a preset-typed action for its resolved concrete
// ActionConfig (helper modules register presets in the core registry).
func resolvePreset(b config.Binding) (config.Binding, error) {
	ac, err := ResolveActionPreset(b.Action.Preset, b.Action.Params)
	if err != nil {
		return b, err
	}
	if ac.Type == config.ActionTypePreset {
		return b, fmt.Errorf("preset %q resolved to another preset — loop", b.Action.Preset)
	}
	if ac.ValueType == "" {
		ac.ValueType = b.Action.ValueType
	}
	b.Action = ac
	return b, nil
}
