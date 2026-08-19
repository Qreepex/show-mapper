// Package osc implements the generic OSC output target (Open Sound Control
// over UDP) — one of possibly many target modules. Module docs: README.md in
// this directory; receiver-specific setup (e.g. grandMA3) is documented in
// the helper module internal/helpers/gma3/README.md.
package osc

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

func init() {
	core.RegisterTarget("osc", core.TypeInfo{
		Name: "OSC (Open Sound Control over UDP) — grandMA3, other show software",
		Options: []core.FieldSpec{
			{Name: "host", Label: "Host / IP", Type: "text", Required: true,
				Help: "IP of the OSC receiver."},
			{Name: "port", Label: "Port", Type: "number", Required: true, Default: 8000,
				Help: "UDP port of the OSC receiver."},
			{Name: "prefix", Label: "Address prefix", Type: "text",
				Help: "Optional prefix prepended to every address (no slashes)."},
			{Name: "localAddress", Label: "Local address (NIC)", Type: "text",
				Help: "Optional: local IPv4 address (NIC) the outgoing socket binds to — for machines with multiple network interfaces. List yours via GET /api/system/interfaces or the Settings hint. Multiple instances can use different NICs at the same time."},
		},
	}, NewTarget)
}

// Target sends OSC messages over UDP, optionally bound to a specific local
// address (NIC) for multi-NIC machines.
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

// NewTarget builds an OSC target from config.
func NewTarget(cfg config.TargetConfig) (core.Target, error) {
	host := strings.TrimSpace(config.OptionString(cfg.Options, "host", ""))
	if host == "" {
		return nil, fmt.Errorf("target %q (osc): options.host is required", cfg.ID)
	}
	port := config.OptionInt(cfg.Options, "port", 8000)
	if port < 1 || port > 65535 {
		return nil, fmt.Errorf("target %q (osc): options.port %d out of range", cfg.ID, port)
	}
	prefix := strings.Trim(config.OptionString(cfg.Options, "prefix", ""), "/")
	var local net.IP
	if s := strings.TrimSpace(config.OptionString(cfg.Options, "localAddress", "")); s != "" {
		ip := net.ParseIP(s)
		if ip == nil || ip.To4() == nil {
			return nil, fmt.Errorf("target %q (osc): options.localAddress %q is not a valid IPv4 address", cfg.ID, s)
		}
		local = ip
	}
	return &Target{
		id:     cfg.ID,
		host:   host,
		port:   port,
		prefix: prefix,
		local:  local,
		st:     core.Status{State: core.StateDisconnected},
	}, nil
}

func (t *Target) ID() string   { return t.id }
func (t *Target) Type() string { return "osc" }

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
	t.st = core.Status{
		State:  core.StateConnected,
		Detail: t.detail(),
	}
	return nil
}

// Send dispatches one resolved action as one OSC message.
// (grandMA3 does not support OSC bundles — see internal/helpers/gma3/README.md.)
func (t *Target) Send(a core.Action) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.client == nil {
		return fmt.Errorf("osc target %q not connected", t.id)
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
			return fmt.Errorf("osc: unsupported argument type %T", arg)
		}
	}
	if err := t.client.Send(msg); err != nil {
		t.st = core.Status{State: core.StateError, Detail: err.Error()}
		return fmt.Errorf("osc send %s: %w", msg.Address, err)
	}
	t.st = core.Status{State: core.StateConnected, Detail: t.detail()}
	return nil
}

func (t *Target) detail() string {
	d := fmt.Sprintf("udp %s:%d", t.host, t.port)
	if t.local != nil {
		d = fmt.Sprintf("udp %s → %s:%d", t.local, t.host, t.port)
	}
	return d
}

// address applies the optional MA-style prefix: prefix "ma" + "/cmd" => "/ma/cmd".
func (t *Target) address(addr string) string {
	if t.prefix == "" {
		return addr
	}
	return "/" + t.prefix + addr
}

func (t *Target) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.client = nil // UDP: nothing to tear down
	t.st = core.Status{State: core.StateDisconnected}
	return nil
}

func (t *Target) Status() core.Status {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.st
}
