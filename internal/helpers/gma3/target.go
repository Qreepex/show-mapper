package gma3

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"

	goosc "github.com/hypebeast/go-osc/osc"

	"github.com/Qreepex/show-mapper/internal/config"
	"github.com/Qreepex/show-mapper/internal/core"
)

// The target half of this helper module: connector type "gma3". It reuses the
// OSC/UDP transport mechanics but stays intentionally self-contained — the
// generic osc module knows nothing about grandMA3 and this module doesn't
// depend on it (both speak OSC over UDP directly).

func init() {
	core.RegisterTarget("gma3", core.TypeInfo{
		Name: "grandMA3 console/onPC (OSC/UDP + action presets)",
		Options: []core.FieldSpec{
			{Name: "host", Label: "Console/onPC IP", Type: "text", Required: true,
				Help: "IP of the grandMA3 station (Network → In & Out → OSC, Preferred IP)."},
			{Name: "port", Label: "Port", Type: "number", Required: true, Default: 8000,
				Help: "Port of the OSC row (same port sends AND receives in MA3)."},
			{Name: "prefix", Label: "Address prefix", Type: "text",
				Help: "Optional; must match the row's Prefix (no slashes)."},
			{Name: "localAddress", Label: "Local address (NIC)", Type: "text",
				Help: "Optional local IPv4 to bind the outgoing socket to (multi-NIC machines)."},
		},
	}, NewTarget)
}

// Target talks OSC to a grandMA3 station.
type Target struct {
	id     string
	host   string
	port   int
	prefix string
	local  net.IP

	mu     sync.Mutex
	st     core.Status
	client *goosc.Client
}

// NewTarget builds a grandMA3 target from config.
func NewTarget(cfg config.TargetConfig) (core.Target, error) {
	host := strings.TrimSpace(config.OptionString(cfg.Options, "host", ""))
	if host == "" {
		return nil, fmt.Errorf("target %q (gma3): options.host is required (console/onPC IP)", cfg.ID)
	}
	port := config.OptionInt(cfg.Options, "port", 8000)
	if port < 1 || port > 65535 {
		return nil, fmt.Errorf("target %q (gma3): options.port %d out of range", cfg.ID, port)
	}
	prefix := strings.Trim(config.OptionString(cfg.Options, "prefix", ""), "/")

	var local net.IP
	if s := strings.TrimSpace(config.OptionString(cfg.Options, "localAddress", "")); s != "" {
		ip := net.ParseIP(s)
		if ip == nil || ip.To4() == nil {
			return nil, fmt.Errorf("target %q (gma3): options.localAddress %q is not a valid IPv4 address", cfg.ID, s)
		}
		local = ip
	}

	return &Target{
		id: cfg.ID, host: host, port: port, prefix: prefix, local: local,
		st: core.Status{State: core.StateDisconnected},
	}, nil
}

func (t *Target) ID() string   { return t.id }
func (t *Target) Type() string { return "gma3" }

func (t *Target) Connect(_ context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	client := goosc.NewClient(t.host, t.port)
	if t.local != nil {
		if err := client.SetLocalAddr(t.local.String(), 0); err != nil {
			t.st = core.Status{State: core.StateError, Detail: err.Error()}
			return fmt.Errorf("binding local address %s: %w", t.local, err)
		}
	}
	t.client = client
	t.st = core.Status{State: core.StateConnected, Detail: t.detail()}
	return nil
}

// Send dispatches one resolved action as one OSC message (MA3 ignores bundles).
func (t *Target) Send(a core.Action) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.client == nil {
		return fmt.Errorf("gma3 target %q not connected", t.id)
	}
	msg := goosc.NewMessage(t.address(a.Address))
	for _, arg := range a.Args {
		switch v := arg.(type) {
		case string:
			msg.Append(v)
		case int32:
			msg.Append(v)
		case float32:
			msg.Append(v)
		default:
			return fmt.Errorf("gma3: unsupported argument type %T", arg)
		}
	}
	if err := t.client.Send(msg); err != nil {
		t.st = core.Status{State: core.StateError, Detail: err.Error()}
		return fmt.Errorf("gma3 send %s: %w", msg.Address, err)
	}
	t.st = core.Status{State: core.StateConnected, Detail: t.detail()}
	return nil
}

func (t *Target) address(addr string) string {
	if t.prefix == "" {
		return addr
	}
	return "/" + t.prefix + addr
}

func (t *Target) detail() string {
	if t.local != nil {
		return fmt.Sprintf("udp %s → %s:%d (grandMA3)", t.local, t.host, t.port)
	}
	return fmt.Sprintf("udp %s:%d (grandMA3)", t.host, t.port)
}

func (t *Target) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.client = nil
	t.st = core.Status{State: core.StateDisconnected}
	return nil
}

func (t *Target) Status() core.Status {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.st
}
