// Package gma3 is the grandMA3 helper *module*: it registers ready-made
// action presets (Go, Go-, Pause, Flash, Temp, On, Off, executor key/fader)
// AND the "gma3" target type (OSC transport with MA3-appropriate defaults) —
// see target.go. Everything grandMA3-specific lives in this directory
// (project rule: core stays console-agnostic; the app runs with or without
// this module).
package gma3

import (
	"fmt"
	"strings"

	"github.com/Qreepex/show-mapper/internal/config"
	"github.com/Qreepex/show-mapper/internal/core"
)

// Address conventions are documented in README.md (console-side OSC setup).
// In short: executor keys/faders are addressed as /Page<P>/(Key|Fader)<E>, and
// free-form keyword commands go to /cmd as a string.

const targetGMA3 = "gma3"

func init() {
	keywordFields := []core.FieldSpec{fieldPage, fieldExecutor}
	for _, p := range []struct{ id, label, help, keyword string }{
		{"gma3.cmd", "grandMA3: keyword command (/cmd)", "Runs any grandMA3 keyword via OSC /cmd (console needs 'Receive Command' enabled on the OSC row).", ""},
		{"gma3.go", "grandMA3: Go+", "", "Go+"},
		{"gma3.goback", "grandMA3: Go- (GoBack)", "", "Go-"},
		{"gma3.pause", "grandMA3: Pause", "", "Pause"},
		{"gma3.temp", "grandMA3: Temp", "", "Temp"},
		{"gma3.on", "grandMA3: On", "", "On"},
		{"gma3.off", "grandMA3: Off", "", "Off"},
	} {
		p := p
		fields := keywordFields
		if p.id == "gma3.cmd" {
			fields = []core.FieldSpec{fieldCommand}
		}
		help := p.help
		if help == "" {
			help = fmt.Sprintf("%s — OSC /cmd %q Page <page>.<executor>.", p.label, p.keyword)
		}
		core.RegisterActionPreset(core.ActionPreset{
			ID: p.id, Source: targetGMA3, Label: p.label, Help: help,
			Fields: fields, TargetTypes: []string{targetGMA3},
		}, func(params map[string]any) (config.ActionConfig, error) {
			if p.id == "gma3.cmd" {
				return cmdPreset(params)
			}
			return keywordPreset(p.keyword, params)
		})
	}

	// Flash needs separate On/Off commands for press/release.
	core.RegisterActionPreset(core.ActionPreset{
		ID: "gma3.flash", Source: targetGMA3, Label: "grandMA3: Flash (momentary)",
		Help:        "Flash On (press) / Flash Off (release) — OSC /cmd \"Flash On Page <page>.<executor>\".",
		Fields:      keywordFields,
		TargetTypes: []string{targetGMA3},
	}, flashPreset)

	core.RegisterActionPreset(core.ActionPreset{
		ID: "gma3.key", Source: targetGMA3, Label: "grandMA3: executor key (press/release)",
		Help:        "Direct key presses on an executor: press (>0) / release (0) — ideal for toggle bindings.",
		Fields:      keywordFields,
		TargetTypes: []string{targetGMA3},
	}, keyPreset)

	core.RegisterActionPreset(core.ActionPreset{
		ID: "gma3.fader", Source: targetGMA3, Label: "grandMA3: executor fader",
		Help:        "Scales a hardware fader/encoder onto an executor fader.",
		Fields:      []core.FieldSpec{fieldPage, fieldExecutor, fieldMin, fieldMax},
		TargetTypes: []string{targetGMA3},
	}, faderPreset)
}

var (
	fieldPage = core.FieldSpec{
		Name: "page", Label: "Page", Type: "text", Required: true,
		Help: "Page number or name (executors always need a page in MA3).",
	}
	fieldExecutor = core.FieldSpec{
		Name: "executor", Label: "Executor", Type: "text", Required: true,
		Help: "Executor number or name on that page (e.g. 201).",
	}
	fieldCommand = core.FieldSpec{
		Name: "command", Label: "Command", Type: "text", Required: true,
		Help: "grandMA3 keyword line, e.g. \"Go+ Page 1.201\".",
	}
	fieldMin = core.FieldSpec{
		Name: "min", Label: "Range min", Type: "number", Default: 0,
	}
	fieldMax = core.FieldSpec{
		Name: "max", Label: "Range max", Type: "number", Default: 100,
		Help: "Mapped onto MA3's FaderRange of that OSC row.",
	}
)

// ---------------------------------------------------------------------------

func cmdPreset(params map[string]any) (config.ActionConfig, error) {
	cmd := strings.TrimSpace(strParam(params, "command"))
	if cmd == "" {
		return config.ActionConfig{}, fmt.Errorf("command is required")
	}
	return config.ActionConfig{Type: config.ActionCommand, Address: "/cmd", Command: cmd}, nil
}

func keywordPreset(keyword string, params map[string]any) (config.ActionConfig, error) {
	page, exec, err := pageExec(params)
	if err != nil {
		return config.ActionConfig{}, err
	}
	return config.ActionConfig{
		Type:    config.ActionCommand,
		Address: "/cmd",
		Command: fmt.Sprintf("%s Page %s.%s", keyword, page, exec),
	}, nil
}

// flashPreset is the factory for gma3.flash — sends Flash On / Flash Off via /cmd.
func flashPreset(params map[string]any) (config.ActionConfig, error) {
	page, exec, err := pageExec(params)
	if err != nil {
		return config.ActionConfig{}, err
	}
	return config.ActionConfig{
		Type:           config.ActionCommand,
		Address:        "/cmd",
		Command:        fmt.Sprintf("Flash On Page %s.%s", page, exec),
		ReleaseCommand: fmt.Sprintf("Flash Off Page %s.%s", page, exec),
	}, nil
}

func keyPreset(params map[string]any) (config.ActionConfig, error) {
	page, exec, err := pageExec(params)
	if err != nil {
		return config.ActionConfig{}, err
	}
	on, off := 1.0, 0.0
	return config.ActionConfig{
		Type: config.ActionValue, Address: fmt.Sprintf("/Page%s/Key%s", page, exec),
		PressValue: &on, ReleaseValue: &off, ValueType: config.ValueTypeInt,
	}, nil
}

func faderPreset(params map[string]any) (config.ActionConfig, error) {
	page, exec, err := pageExec(params)
	if err != nil {
		return config.ActionConfig{}, err
	}
	min, max := 0.0, 100.0
	if v, ok := numParam(params, "min"); ok {
		min = v
	}
	if v, ok := numParam(params, "max"); ok {
		max = v
	}
	if min >= max {
		return config.ActionConfig{}, fmt.Errorf("min must be < max")
	}
	return config.ActionConfig{
		Type: config.ActionFader, Address: fmt.Sprintf("/Page%s/Fader%s", page, exec),
		Range: &[2]float64{min, max}, ValueType: config.ValueTypeInt,
	}, nil
}

// ---------------------------------------------------------------------------

// pageExec accepts numbers or names (JSON → float64, YAML → int, UI → string).
func pageExec(params map[string]any) (page, exec string, err error) {
	page = strParam(params, "page")
	exec = strParam(params, "executor")
	if page == "" || exec == "" {
		return "", "", fmt.Errorf("page and executor are required")
	}
	return page, exec, nil
}

func strParam(params map[string]any, key string) string {
	switch v := params[key].(type) {
	case string:
		return strings.TrimSpace(v)
	case int:
		return fmt.Sprint(v)
	case int64:
		return fmt.Sprint(v)
	case float64:
		if v == float64(int64(v)) {
			return fmt.Sprint(int64(v))
		}
		return fmt.Sprint(v)
	}
	return ""
}

func numParam(params map[string]any, key string) (float64, bool) {
	switch v := params[key].(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case string:
		if v == "" {
			return 0, false
		}
		var f float64
		if _, err := fmt.Sscanf(v, "%g", &f); err == nil {
			return f, true
		}
	}
	return 0, false
}
