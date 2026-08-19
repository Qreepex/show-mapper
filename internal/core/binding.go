package core

import (
	"math"

	"github.com/yourorg/showbridge/internal/config"
)

// This file resolves config.Bindings into concrete Actions.
// It lives in core (not in config) because the Action type lives here and
// config may not import core (import cycle). All trigger/mode *matching*
// happens in conductor.go; these helpers only resolve payloads.

// PressAction resolves the payload for a "press" (or hold-fire, toggle-on).
// ok == false means "configured to do nothing".
func pressAction(b config.Binding) (Action, bool) {
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

// ReleaseAction resolves the payload for a "release" (or toggle-off in toggle mode).
func releaseAction(b config.Binding) (Action, bool) {
	base := Action{BindingID: b.Key(), TargetID: b.Target, Kind: ActionKind(b.Action.Type), Address: b.Action.Address}
	switch b.Action.Type {
	case config.ActionCommand:
		if b.Action.ReleaseCommand == "" {
			return Action{}, false
		}
		base.Args = []any{b.Action.ReleaseCommand}
		return base, true
	case config.ActionValue:
		if b.Action.ReleaseValue == nil {
			return Action{}, false
		}
		base.Args = []any{numericArg(*b.Action.ReleaseValue, b.Action.ValueType)}
		return base, true
	}
	return Action{}, false
}

// ValueAction resolves the payload for an analog "value" trigger:
// the normalized source value (0..1) is scaled into the configured range.
func valueAction(b config.Binding, v float64) (Action, bool) {
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
