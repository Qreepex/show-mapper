// Package config defines the show-mapper configuration file format,
// loading, validation and persistence.
//
// The config is a single YAML file (default: ./show-mapper.yaml or
// $USER_CONFIG_DIR/show-mapper/config.yaml). It is edited either by hand
// or through the web UI (which PUTs to /api/config). The backend hot-reloads
// connectors after a successful save.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// SchemaVersion is the current config schema version. Bump when the format
// changes incompatibly and provide a migration in Migrate().
const SchemaVersion = 1

// idRule defines valid connector / binding IDs: lowercase DNS-label-ish.
// Keep in sync with docs/architecture.md (Naming conventions) and the web UI.
var idRule = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)

// Config is the root of the configuration file.
type Config struct {
	Version int        `yaml:"version" json:"version"`
	HTTP    HTTPConfig `yaml:"http" json:"http"`
	// Updates, if set, configures the self-update checker
	// (see internal/updater). Disabled when nil or repo empty.
	Updates *UpdatesConfig `yaml:"updates,omitempty" json:"updates,omitempty"`

	// Profiles holds USER-DEFINED device profiles ("custom boards").
	// Built-in profiles (e.g. MIDI profiles apc-mini-mk2, apc-mini, mpk-mini-mk3) ship with the binary;
	// this section lets users describe any other hardware — see
	// docs/midi-devices.md#custom-boards and the UI Boards section.
	Profiles []ProfileConfig `yaml:"profiles,omitempty" json:"profiles,omitempty"`
	Sources  []SourceConfig  `yaml:"sources" json:"sources"`
	Targets  []TargetConfig  `yaml:"targets" json:"targets"`
	Bindings []Binding       `yaml:"bindings" json:"bindings"`
}

// UpdatesConfig configures update checking (GitHub releases of `repo`).
type UpdatesConfig struct {
	Repo      string `yaml:"repo" json:"repo"`                     // "owner/name"
	AutoCheck bool   `yaml:"autoCheck,omitempty" json:"autoCheck"` // check on startup
}

var repoRule = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)

// ProfileConfig describes a user-defined device profile.
//
// Example (MIDI):
//
//	profiles:
//	  - id: my-x-touch
//	    type: midi
//	    name: Behringer X-Touch Compact
//	    match: ["X-TOUCH"]
//	    led: { style: onOff }
//	    controls:
//	      - { id: pad-1, kind: pad,    label: "Pad 1",   note: 36, hasLED: true }
//	      - { id: fader-1, kind: fader, label: "Fader 1", cc: 1 }
type ProfileConfig struct {
	ID   string `yaml:"id" json:"id"`
	Type string `yaml:"type" json:"type"` // connector type this profile belongs to, e.g. "midi"
	Name string `yaml:"name" json:"name"`
	// Match holds case-insensitive port/device-name substrings used for
	// auto-detection when a source omits `profile`.
	Match    []string         `yaml:"match,omitempty" json:"match,omitempty"`
	LED      ProfileLED       `yaml:"led,omitempty" json:"led,omitempty"`
	Controls []ProfileControl `yaml:"controls" json:"controls"`
}

// ProfileLED describes how a custom board's LEDs are driven (MIDI).
type ProfileLED struct {
	// Style: none (default) | onOff | velocity | apc2-rgb
	Style string `yaml:"style,omitempty" json:"style,omitempty"`
	// OnVelocity is the "on" velocity for style onOff (default 127).
	OnVelocity int `yaml:"onVelocity,omitempty" json:"onVelocity,omitempty"`
	// Colors maps color names to velocity values for style velocity
	// (empty = APC-mini-1 scheme: green=1, red=3, yellow=5).
	Colors map[string]int `yaml:"colors,omitempty" json:"colors,omitempty"`
}

// LEDStyles lists the valid ProfileLED styles.
var LEDStyles = []string{"none", "onOff", "velocity", "apc2-rgb"}

// ProfileControl is one control of a user-defined profile.
type ProfileControl struct {
	ID    string `yaml:"id" json:"id"`
	Label string `yaml:"label" json:"label"`
	Kind  string `yaml:"kind" json:"kind"` // pad | button | fader | encoder
	// Row/Col are optional grid coordinates for pad rendering in the UI.
	Row *int `yaml:"row,omitempty" json:"row,omitempty"`
	Col *int `yaml:"col,omitempty" json:"col,omitempty"`
	// MIDI addressing: exactly one of Note / CC.
	Note *int `yaml:"note,omitempty" json:"note,omitempty"`
	CC   *int `yaml:"cc,omitempty" json:"cc,omitempty"`
	// HasLED marks the control as lightable; LedNote overrides the LED note
	// number if it differs from Note (boards with split LED addressing).
	HasLED  bool `yaml:"hasLED,omitempty" json:"hasLED,omitempty"`
	LEDNote *int `yaml:"ledNote,omitempty" json:"ledNote,omitempty"`
}

// ControlKinds lists valid ProfileControl kinds.
var ControlKinds = []string{"pad", "button", "fader", "encoder"}

// HTTPConfig configures the embedded web server (UI + API + WebSocket).
type HTTPConfig struct {
	// Listen is the address the web UI binds to. Defaults to 127.0.0.1:8484.
	// Set to "0.0.0.0:8484" (or ":8484") to expose it on the show network.
	// There is no authentication yet — see docs/architecture.md (Security).
	Listen string `yaml:"listen" json:"listen"`
	// OpenBrowser controls whether the UI is opened in the system browser at
	// startup (default true). `show-mapper serve -no-browser` overrides.
	OpenBrowser *bool `yaml:"openBrowser,omitempty" json:"openBrowser,omitempty"`
}

// ShouldOpenBrowser resolves the default (true unless explicitly false).
func (h HTTPConfig) ShouldOpenBrowser() bool {
	return h.OpenBrowser == nil || *h.OpenBrowser
}

// SourceConfig describes one physical/logical event source instance,
// e.g. one connected APC mini mk2.
type SourceConfig struct {
	ID   string `yaml:"id" json:"id"`     // instance ID, e.g. "wing-left"
	Type string `yaml:"type" json:"type"` // connector type, e.g. "midi"
	// Profile selects the device layout for connectors that need one
	// (e.g. MIDI device profiles "apc-mini-mk2"). See docs/midi-devices.md.
	Profile string `yaml:"profile,omitempty" json:"profile,omitempty"`
	// Options holds connector-type-specific settings (decoded by the connector).
	Options map[string]any `yaml:"options,omitempty" json:"options,omitempty"`
}

// TargetConfig describes one output target instance, e.g. a grandMA3 console.
type TargetConfig struct {
	ID      string         `yaml:"id" json:"id"`
	Type    string         `yaml:"type" json:"type"` // e.g. "osc", future: "artnet", "sacn", ...
	Options map[string]any `yaml:"options,omitempty" json:"options,omitempty"`
}

// Binding maps an event on a source control to an action on a target.
type Binding struct {
	// ID is optional; if empty a stable ID is derived from source/control/trigger.
	ID string `yaml:"id,omitempty" json:"id,omitempty"`

	Source  string `yaml:"source" json:"source"`   // SourceConfig.ID
	Control string `yaml:"control" json:"control"` // control ID, e.g. "pad-3-4"; see the source's profile
	Trigger string `yaml:"trigger" json:"trigger"` // pressed | released | hold | value
	// HoldMs is the press duration (milliseconds) after which a "hold" binding fires.
	// Only used when Trigger == "hold". Default: 500.
	HoldMs int `yaml:"holdMs,omitempty" json:"holdMs,omitempty"`
	// Mode: "momentary" (default) or "toggle".
	// - momentary: OnPress on press, OnRelease on release.
	// - toggle:    press toggles internal state; "on" sends OnPress/first value,
	//              "off" sends OnRelease/second value. Also drives pad LEDs.
	Mode   string       `yaml:"mode,omitempty" json:"mode,omitempty"`
	Target string       `yaml:"target" json:"target"` // TargetConfig.ID
	Action ActionConfig `yaml:"action" json:"action"`
	// LED optionally configures pad/button LED feedback for toggle bindings
	// (only effective if the source device has LEDs, e.g. APC mini mk2).
	LED *LEDConfig `yaml:"led,omitempty" json:"led,omitempty"`
}

// LEDConfig describes the desired LED look for the "on" state of a toggle.
type LEDConfig struct {
	Color string `yaml:"color,omitempty" json:"color"` // palette name: red, green, amber, ... (device-defined)
	Mode  string `yaml:"mode,omitempty" json:"mode"`   // "on" (default), "blink", "pulse"
}

// LEDColor returns the configured LED color or the default ("green").
func (b Binding) LEDColor() string {
	if b.LED != nil && b.LED.Color != "" {
		return b.LED.Color
	}
	return "green"
}

// LEDMode returns the configured LED mode or the default ("on").
func (b Binding) LEDMode() string {
	if b.LED != nil && b.LED.Mode != "" {
		return b.LED.Mode
	}
	return "on"
}

// ActionConfig describes what to send to a target. Fields used depend on Type.
type ActionConfig struct {
	Type string `yaml:"type" json:"type"` // command | value | fader | preset

	// Address is the target-side address selector.
	// For target "osc" this is the OSC address, e.g. "/cmd" or "/Page1/Fader201".
	// Not used for Type == "preset" (the preset resolves the address).
	Address string `yaml:"address" json:"address"`

	// Type "command": free-form command strings (e.g. grandMA3 keyword syntax).
	Command        string `yaml:"command,omitempty" json:"command,omitempty"`
	ReleaseCommand string `yaml:"releaseCommand,omitempty" json:"releaseCommand,omitempty"`

	// Type "value": fixed values sent on press (and optionally on release).
	PressValue   *float64 `yaml:"pressValue,omitempty" json:"pressValue,omitempty"`
	ReleaseValue *float64 `yaml:"releaseValue,omitempty" json:"releaseValue,omitempty"`

	// Type "preset": helper-module template, resolved at dispatch. Params feed
	// the preset's fields (see /api/meta presets and internal/helpers/<x>).
	Preset string         `yaml:"preset,omitempty" json:"preset,omitempty"`
	Params map[string]any `yaml:"params,omitempty" json:"params,omitempty"`

	// Type "fader": the source control's value (0..1) is scaled into Range.
	Range *[2]float64 `yaml:"range,omitempty" json:"range,omitempty"`

	// ValueType selects the wire representation for numeric payloads:
	// "int" (int32) or "float" (float32). Default "int" — grandMA3 faders
	// expect integers, see internal/helpers/gma3/README.md.
	ValueType string `yaml:"valueType,omitempty" json:"valueType,omitempty"`
}

// Key returns the stable identity used for runtime state (toggles, timers, UI).
func (b Binding) Key() string {
	if b.ID != "" {
		return b.ID
	}
	return b.Source + "/" + b.Control + "/" + b.Trigger
}

const (
	TriggerPressed  = "pressed"
	TriggerReleased = "released"
	TriggerHold     = "hold"
	TriggerValue    = "value"
)

// Triggers lists the valid binding triggers (exposed to the UI via /api/meta).
var Triggers = []string{TriggerPressed, TriggerReleased, TriggerHold, TriggerValue}

const (
	ModeMomentary = "momentary"
	ModeToggle    = "toggle"
)

// Modes lists the valid binding modes.
var Modes = []string{ModeMomentary, ModeToggle}

const (
	ActionCommand = "command"
	ActionValue   = "value"
	ActionFader   = "fader"
	// ActionPreset is a helper-module action stored by reference
	// (preset id + params), resolved at dispatch time via the preset registry.
	ActionTypePreset = "preset"
)

// ActionTypes lists the valid action types.
var ActionTypes = []string{ActionCommand, ActionValue, ActionFader, ActionTypePreset}

const (
	ValueTypeInt   = "int"
	ValueTypeFloat = "float"
)

// Default returns a minimal working configuration.
func Default() Config {
	return Config{
		Version: SchemaVersion,
		HTTP:    HTTPConfig{Listen: "127.0.0.1:8484"},
	}
}

// Candidates returns config file locations in probe order ($SHOWMAPPER_CONFIG
// wins over everything).
func Candidates() []string {
	out := []string{}
	if p := os.Getenv("SHOWMAPPER_CONFIG"); p != "" {
		out = append(out, p)
	}
	if wd, err := os.Getwd(); err == nil {
		out = append(out, filepath.Join(wd, "show-mapper.yaml"))
	}
	// portable-app mode: the binary's own directory (double-click runs)
	if exe, err := os.Executable(); err == nil {
		if real, err := filepath.EvalSymlinks(exe); err == nil {
			exe = real
		}
		out = append(out, filepath.Join(filepath.Dir(exe), "show-mapper.yaml"))
	}
	if dir, err := os.UserConfigDir(); err == nil {
		out = append(out, filepath.Join(dir, "show-mapper", "config.yaml"))
	}
	return out
}

// File is a loaded config plus thread-safe accessors.
// Path is where Save() writes (set by Load/LoadPath).
type File struct {
	mu   sync.RWMutex
	cfg  Config
	Path string
}

// Get returns the current config (copy).
func (f *File) Get() Config {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.cfg
}

// Set replaces the config.
func (f *File) Set(c Config) {
	f.mu.Lock()
	f.cfg = c
	f.mu.Unlock()
}

// ErrNotFound marks "no config file anywhere" (trigger for auto-creation).
var ErrNotFound = errors.New("no config file found")

// Load finds and parses the config file. Returns ErrNotFound (wrapped) when
// no candidate exists; parse failures are regular errors.
func Load() (*File, error) {
	for _, p := range Candidates() {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		cfg, err := Parse(data)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", p, err)
		}
		return &File{cfg: cfg, Path: p}, nil
	}
	return nil, fmt.Errorf("%w (looked in: %s)", ErrNotFound, strings.Join(Candidates(), ", "))
}

// LoadPath parses a specific config file (used by the -config flag).
func LoadPath(path string) (*File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg, err := Parse(data)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return &File{cfg: cfg, Path: path}, nil
}

// starterHeader is prepended to auto-generated starter configs.
const starterHeader = `# show-mapper configuration — generated automatically on first start.
# Open the web UI (printed at startup, default http://127.0.0.1:8484) to add
# boards / targets / mappings. Hand edits of this file are hot-reloaded.
# Full annotated example: show-mapper.example.yaml

`

// PreferredCreatePath picks where to auto-create a starter config:
// $SHOWMAPPER_CONFIG, else ./show-mapper.yaml if the working dir is writable,
// else the binary's directory, else the OS user-config dir.
func PreferredCreatePath() string {
	if env := os.Getenv("SHOWMAPPER_CONFIG"); env != "" {
		return env
	}
	if wd, err := os.Getwd(); err == nil && dirWritable(wd) {
		return filepath.Join(wd, "show-mapper.yaml")
	}
	if exe, err := os.Executable(); err == nil {
		if real, err := filepath.EvalSymlinks(exe); err == nil {
			exe = real
		}
		if dir := filepath.Dir(exe); dirWritable(dir) {
			return filepath.Join(dir, "show-mapper.yaml")
		}
	}
	if dir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(dir, "show-mapper", "config.yaml")
	}
	return "show-mapper.yaml"
}

// CreateDefault writes a minimal starter config (version + http defaults —
// zero connectors, ready to be filled via the UI) to path and returns it.
func CreateDefault(path string) (*File, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	cfg := Default()
	data, err := MarshalYAML(&cfg)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, append([]byte(starterHeader), data...), 0o600); err != nil {
		return nil, err
	}
	return &File{cfg: cfg, Path: path}, nil
}

func dirWritable(dir string) bool {
	f, err := os.CreateTemp(dir, ".show-mapper-probe-*")
	if err != nil {
		return false
	}
	name := f.Name()
	_ = f.Close()
	_ = os.Remove(name)
	return true
}

// Normalize applies defaults and validates a config that was NOT decoded
// via Parse (e.g. JSON from the REST API). Mutates c.
func Normalize(c *Config) error {
	c.applyDefaults()
	return c.Validate()
}

// Parse decodes and normalizes+validates a config document.
func Parse(data []byte) (Config, error) {
	var cfg Config
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&cfg); err != nil {
		return cfg, err
	}
	if err := Normalize(&cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// Save writes the config atomically (temp file + rename) to f.Path.
func (f *File) Save() error {
	f.mu.RLock()
	cfg := f.cfg
	f.mu.RUnlock()
	cfg.Version = SchemaVersion
	data, err := MarshalYAML(&cfg)
	if err != nil {
		return err
	}
	header := []byte("# show-mapper configuration — see docs/architecture.md and show-mapper.example.yaml\n")
	data = append(header, data...)

	tmp := f.Path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	backup := f.Path + ".bak"
	_ = os.Remove(backup)
	if _, err := os.Stat(f.Path); err == nil {
		// Best-effort backup of the previous file (also makes the swap work on Windows).
		if err := os.Rename(f.Path, backup); err != nil {
			_ = os.Remove(tmp)
			return fmt.Errorf("backing up old config: %w", err)
		}
	}
	if err := os.Rename(tmp, f.Path); err != nil {
		// try to roll back
		_ = os.Rename(backup, f.Path)
		return err
	}
	_ = os.Remove(backup)
	return nil
}

func (c *Config) applyDefaults() {
	if c.HTTP.Listen == "" {
		c.HTTP.Listen = "127.0.0.1:8484"
	}
	for i := range c.Bindings {
		b := &c.Bindings[i]
		if b.Mode == "" {
			b.Mode = ModeMomentary
		}
		if b.Trigger == TriggerHold && b.HoldMs <= 0 {
			b.HoldMs = 500
		}
		if b.Action.ValueType == "" {
			b.Action.ValueType = ValueTypeInt
		}
	}
}

// Validate performs structural validation. It does NOT check connector types
// or profile names — those live in the registry and are verified at startup;
// unknown types produce per-instance errors instead of a fatal config error.
func (c Config) Validate() error {
	var errs []error

	if c.Version > SchemaVersion {
		errs = append(errs, fmt.Errorf("config version %d is newer than supported %d", c.Version, SchemaVersion))
	}
	if c.HTTP.Listen == "" {
		errs = append(errs, errors.New("http.listen must not be empty"))
	}
	if c.Updates != nil && c.Updates.Repo != "" && !repoRule.MatchString(c.Updates.Repo) {
		errs = append(errs, fmt.Errorf("updates.repo %q must look like \"owner/name\"", c.Updates.Repo))
	}

	srcIDs := map[string]bool{}
	for _, s := range c.Sources {
		if !idRule.MatchString(s.ID) {
			errs = append(errs, fmt.Errorf("source %q: id must match %s", s.ID, idRule))
		}
		if s.Type == "" {
			errs = append(errs, fmt.Errorf("source %q: type is required", s.ID))
		}
		if srcIDs[s.ID] {
			errs = append(errs, fmt.Errorf("duplicate source id %q", s.ID))
		}
		srcIDs[s.ID] = true
	}

	profIDs := map[string]bool{}
	for _, p := range c.Profiles {
		where := fmt.Sprintf("profile %q", p.ID)
		if !idRule.MatchString(p.ID) {
			errs = append(errs, fmt.Errorf("%s: id must match %s", where, idRule))
		}
		if p.Type == "" {
			errs = append(errs, fmt.Errorf("%s: type is required (e.g. \"midi\")", where))
		}
		if profIDs[p.ID] {
			errs = append(errs, fmt.Errorf("%s: duplicate profile id", where))
		}
		profIDs[p.ID] = true
		if len(p.Controls) == 0 {
			errs = append(errs, fmt.Errorf("%s: at least one control is required", where))
		}
		if p.LED.Style != "" && !contains(LEDStyles, p.LED.Style) {
			errs = append(errs, fmt.Errorf("%s: led.style must be one of %s", where, strings.Join(LEDStyles, ", ")))
		}
		ctrlIDs := map[string]bool{}
		for _, ct := range p.Controls {
			cw := fmt.Sprintf("%s control %q", where, ct.ID)
			if !idRule.MatchString(ct.ID) {
				errs = append(errs, fmt.Errorf("%s: id must match %s", cw, idRule))
			}
			if ctrlIDs[ct.ID] {
				errs = append(errs, fmt.Errorf("%s: duplicate control id", cw))
			}
			ctrlIDs[ct.ID] = true
			if !contains(ControlKinds, ct.Kind) {
				errs = append(errs, fmt.Errorf("%s: kind must be one of %s", cw, strings.Join(ControlKinds, ", ")))
			}
			switch {
			case ct.Note == nil && ct.CC == nil:
				errs = append(errs, fmt.Errorf("%s: one of note/cc is required", cw))
			case ct.Note != nil && ct.CC != nil:
				errs = append(errs, fmt.Errorf("%s: set either note or cc, not both", cw))
			}
			for name, v := range map[string]*int{"note": ct.Note, "cc": ct.CC, "ledNote": ct.LEDNote} {
				if v != nil && (*v < 0 || *v > 127) {
					errs = append(errs, fmt.Errorf("%s: %s %d out of MIDI range 0..127", cw, name, *v))
				}
			}
		}
	}

	tgtIDs := map[string]bool{}
	for _, t := range c.Targets {
		if !idRule.MatchString(t.ID) {
			errs = append(errs, fmt.Errorf("target %q: id must match %s", t.ID, idRule))
		}
		if t.Type == "" {
			errs = append(errs, fmt.Errorf("target %q: type is required", t.ID))
		}
		if tgtIDs[t.ID] {
			errs = append(errs, fmt.Errorf("duplicate target id %q", t.ID))
		}
		tgtIDs[t.ID] = true
	}

	seenKeys := map[string]bool{}
	for i, b := range c.Bindings {
		where := fmt.Sprintf("binding[%d] (%s)", i, b.Key())
		if !srcIDs[b.Source] {
			errs = append(errs, fmt.Errorf("%s: unknown source %q", where, b.Source))
		}
		if !tgtIDs[b.Target] {
			errs = append(errs, fmt.Errorf("%s: unknown target %q", where, b.Target))
		}
		if b.Control == "" {
			errs = append(errs, fmt.Errorf("%s: control is required", where))
		}
		if !contains(Triggers, b.Trigger) {
			errs = append(errs, fmt.Errorf("%s: trigger must be one of %s", where, strings.Join(Triggers, ", ")))
		}
		if !contains(Modes, b.Mode) && b.Mode != "" {
			errs = append(errs, fmt.Errorf("%s: mode must be one of %s", where, strings.Join(Modes, ", ")))
		}
		if b.ID != "" && !idRule.MatchString(b.ID) {
			errs = append(errs, fmt.Errorf("%s: id must match %s", where, idRule))
		}
		if b.HoldMs < 0 || b.HoldMs > 60_000 {
			errs = append(errs, fmt.Errorf("%s: holdMs out of range", where))
		}
		errs = append(errs, b.Action.validate(where)...)
		k := b.Key()
		if seenKeys[k] {
			errs = append(errs, fmt.Errorf("%s: duplicate binding key %q", where, k))
		}
		seenKeys[k] = true
	}

	return errors.Join(errs...)
}

func (a ActionConfig) validate(where string) []error {
	var errs []error
	if !contains(ActionTypes, a.Type) {
		errs = append(errs, fmt.Errorf("%s: action.type must be one of %s", where, strings.Join(ActionTypes, ", ")))
	}
	if a.Type == ActionTypePreset {
		if a.Preset == "" {
			errs = append(errs, fmt.Errorf("%s: action.preset required for type=preset", where))
		}
		return errs
	}
	if !strings.HasPrefix(a.Address, "/") {
		errs = append(errs, fmt.Errorf("%s: action.address must start with \"/\"", where))
	}
	switch a.Type {
	case ActionCommand:
		if a.Command == "" {
			errs = append(errs, fmt.Errorf("%s: action.command required for type=command", where))
		}
	case ActionValue:
		if a.PressValue == nil {
			errs = append(errs, fmt.Errorf("%s: action.pressValue required for type=value", where))
		}
	case ActionFader:
		if a.Range == nil || a.Range[0] >= a.Range[1] {
			errs = append(errs, fmt.Errorf("%s: action.range [min,max] with min < max required for type=fader", where))
		}
	}
	if a.ValueType != "" && a.ValueType != ValueTypeInt && a.ValueType != ValueTypeFloat {
		errs = append(errs, fmt.Errorf("%s: action.valueType must be %q or %q", where, ValueTypeInt, ValueTypeFloat))
	}
	return errs
}

// MarshalYAML renders a config in the canonical hand-editable style
// (2-space indents — the same style as show-mapper.example.yaml, so
// hand-appended entries and app-saved files don't clash).
func MarshalYAML(c *Config) ([]byte, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(c); err != nil {
		return nil, err
	}
	return buf.Bytes(), enc.Close()
}

// Equal reports whether two configs marshal to the same YAML.
// Used by the config-file watcher to skip self-inflicted saves.
func Equal(a, b Config) bool {
	ya, ea := MarshalYAML(&a)
	yb, eb := MarshalYAML(&b)
	return ea == nil && eb == nil && bytes.Equal(ya, yb)
}

func contains(list []string, s string) bool {
	return sort.SearchStrings(list, s) < len(list) && list[sort.SearchStrings(list, s)] == s
}

func init() {
	// keep the lookup lists sorted for contains()
	for _, l := range [][]string{Triggers, Modes, ActionTypes, LEDStyles, ControlKinds} {
		sort.Strings(l)
	}
}

// OptionString reads a string option with default.
func OptionString(opts map[string]any, name, def string) string {
	if v, ok := opts[name]; ok {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	return def
}

// OptionInt reads an integer option with default (YAML numbers arrive as int,
// JSON numbers as float64).
func OptionInt(opts map[string]any, name string, def int) int {
	if v, ok := opts[name]; ok {
		switch n := v.(type) {
		case int:
			return n
		case int64:
			return int(n)
		case float64:
			return int(n)
		}
	}
	return def
}
