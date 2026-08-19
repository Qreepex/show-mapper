package core

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/yourorg/showbridge/internal/config"
)

// ---------------------------------------------------------------------------
// Conductor: wires sources -> bindings -> targets.
// ---------------------------------------------------------------------------

// The Conductor owns all connector instances, matches source events against
// bindings and dispatches actions to targets. It is driven entirely by the
// config; Reload() performs a full restart of all instances (simple and
// correct for human-edited, low-churn configs).
type Conductor struct {
	sink Sink

	mu         sync.Mutex
	cfg        config.Config
	sources    map[string]Source
	targets    map[string]Target
	toggles    map[string]bool             // binding key -> toggle state
	holdTimers map[string]*time.Timer      // binding key -> pending hold timer
	bindIndex  map[string][]config.Binding // "sourceID\x00controlID" -> bindings

	ctx    context.Context
	cancel context.CancelFunc
	// parentCtx is the ctx passed to Run; kept so Reload() can recreate
	// c.ctx with the same parent (otherwise instances survive shutdown).
	parentCtx context.Context
	wg        sync.WaitGroup

	reconnectEvery time.Duration
}

// NewConductor builds a Conductor. Call Run() to start dispatching.
func NewConductor(cfg config.Config, sink Sink) *Conductor {
	if sink == nil {
		sink = NopSink{}
	}
	return &Conductor{
		sink:           sink,
		cfg:            cfg,
		sources:        map[string]Source{},
		targets:        map[string]Target{},
		toggles:        map[string]bool{},
		holdTimers:     map[string]*time.Timer{},
		bindIndex:      map[string][]config.Binding{},
		reconnectEvery: 5 * time.Second,
	}
}

// Config returns the current config.
func (c *Conductor) Config() config.Config {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.cfg
}

// Run starts all instances and blocks until ctx is cancelled.
func (c *Conductor) Run(ctx context.Context) {
	c.parentCtx = ctx
	c.ctx, c.cancel = context.WithCancel(ctx)
	c.mu.Lock()
	c.rebuildLocked()
	c.mu.Unlock()

	<-ctx.Done()
	c.shutdownInstances()
	slog.Info("conductor stopped")
}

// Stop cancels the internal context (used after Reload-less shutdown).
func (c *Conductor) Stop() {
	if c.cancel != nil {
		c.cancel()
	}
}

// Reload swaps the config and restarts all connector instances.
func (c *Conductor) Reload(cfg config.Config) error {
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.shutdownInstancesLocked()
	c.cfg = cfg
	c.toggles = map[string]bool{}
	c.rebuildLocked()
	return nil
}

func (c *Conductor) rebuildLocked() {
	c.sources = map[string]Source{}
	c.targets = map[string]Target{}
	c.bindIndex = map[string][]config.Binding{}
	if c.ctx == nil {
		// Run() hasn't been called yet (e.g. Reload on a fresh conductor).
		c.parentCtx = context.Background()
		c.ctx, c.cancel = context.WithCancel(c.parentCtx)
	}

	for _, sc := range c.cfg.Sources {
		src, err := NewSource(sc, SourceBuildContext{Profiles: c.cfg.Profiles})
		if err != nil {
			slog.Error("create source failed", "id", sc.ID, "type", sc.Type, "err", err)
			continue
		}
		c.sources[sc.ID] = src
	}
	for _, tc := range c.cfg.Targets {
		tgt, err := NewTarget(tc)
		if err != nil {
			slog.Error("create target failed", "id", tc.ID, "type", tc.Type, "err", err)
			continue
		}
		c.targets[tc.ID] = tgt
	}
	for _, b := range c.cfg.Bindings {
		k := indexKey(b.Source, b.Control)
		c.bindIndex[k] = append(c.bindIndex[k], b)
	}

	for id, src := range c.sources {
		c.wg.Add(1)
		go c.runSourceInstance(id, src)
	}
	for id, tgt := range c.targets {
		c.wg.Add(1)
		go c.runTargetInstance(id, tgt)
	}
	slog.Info("conductor (re)built",
		"sources", len(c.sources), "targets", len(c.targets), "bindings", len(c.cfg.Bindings))
}

// runSourceInstance connects (with retry/backoff) and pumps events until
// the conductor is shut down.
func (c *Conductor) runSourceInstance(id string, src Source) {
	defer c.wg.Done()
	log := slog.With("source", id)

	setStatus := func(st Status) {
		c.sink.Broadcast("connector.status", ConnectorStatus{
			ID: id, Kind: "source", Type: src.Type(), Status: st,
		})
	}

	setStatus(Status{State: StateConnecting})
	for {
		if err := src.Connect(c.ctx); err != nil {
			setStatus(Status{State: StateError, Detail: err.Error()})
			if IsPermanent(err) {
				log.Error("permanent connect failure (no retry)", "err", err)
				<-c.ctx.Done()
				return
			}
			log.Warn("connect failed, retrying", "err", err, "retryIn", c.reconnectEvery)
			select {
			case <-c.ctx.Done():
				return
			case <-time.After(c.reconnectEvery):
				continue
			}
		}
		break
	}
	log.Info("source connected")
	setStatus(src.Status())

	events := src.Events()
	for {
		select {
		case <-c.ctx.Done():
			return
		case ev, ok := <-events:
			if !ok {
				return
			}
			c.handleEvent(ev, src)
		}
	}
}

func (c *Conductor) runTargetInstance(id string, tgt Target) {
	defer c.wg.Done()
	log := slog.With("target", id)

	c.sink.Broadcast("connector.status", ConnectorStatus{
		ID: id, Kind: "target", Type: tgt.Type(), Status: Status{State: StateConnecting},
	})
	for {
		if err := tgt.Connect(c.ctx); err != nil {
			c.sink.Broadcast("connector.status", ConnectorStatus{
				ID: id, Kind: "target", Type: tgt.Type(),
				Status: Status{State: StateError, Detail: err.Error()},
			})
			if IsPermanent(err) {
				log.Error("permanent connect failure (no retry)", "err", err)
				<-c.ctx.Done()
				return
			}
			log.Warn("connect failed, retrying", "err", err, "retryIn", c.reconnectEvery)
			select {
			case <-c.ctx.Done():
				return
			case <-time.After(c.reconnectEvery):
				continue
			}
		}
		break
	}
	log.Info("target connected")
	c.sink.Broadcast("connector.status", ConnectorStatus{
		ID: id, Kind: "target", Type: tgt.Type(), Status: tgt.Status(),
	})

	<-c.ctx.Done()
}

// ConnectorStatus is broadcast to the UI whenever an instance changes state.
type ConnectorStatus struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"` // "source" | "target"
	Type   string `json:"type"`
	Status Status `json:"status"`
}

// SnapshotConn is the per-instance part of Snapshot().
type SnapshotConn struct {
	ID       string    `json:"id"`
	Kind     string    `json:"kind"`
	Type     string    `json:"type"`
	Status   Status    `json:"status"`
	Controls []Control `json:"controls,omitempty"` // sources only
}

// Snapshot returns the current runtime view for new WebSocket clients / REST.
func (c *Conductor) Snapshot() []SnapshotConn {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]SnapshotConn, 0, len(c.sources)+len(c.targets))
	for id, s := range c.sources {
		out = append(out, SnapshotConn{ID: id, Kind: "source", Type: s.Type(), Status: s.Status(), Controls: s.Controls()})
	}
	for id, t := range c.targets {
		out = append(out, SnapshotConn{ID: id, Kind: "target", Type: t.Type(), Status: t.Status()})
	}
	return out
}

// ---------------------------------------------------------------------------
// Event dispatch
// ---------------------------------------------------------------------------

func indexKey(sourceID, control string) string { return sourceID + "\x00" + control }

// handleEvent matches an event against bindings and dispatches actions.
// It never blocks a source's event goroutine for long: sends are quick
// (UDP); everything is broadcast to the UI for the live ticker.
func (c *Conductor) handleEvent(ev Event, src Source) {
	c.sink.Broadcast("source.event", ev)

	c.mu.Lock()
	bindings := append([]config.Binding(nil), c.bindIndex[indexKey(ev.SourceID, ev.Control)]...)
	c.mu.Unlock()

	for _, b := range bindings {
		switch {
		case b.Trigger == config.TriggerPressed && ev.Kind == EventPressed:
			c.dispatchPress(b, src)

		case b.Trigger == config.TriggerReleased && ev.Kind == EventReleased:
			c.dispatchRelease(b)

		case b.Trigger == config.TriggerHold && ev.Kind == EventPressed:
			c.armHold(b)

		case b.Trigger == config.TriggerHold && ev.Kind == EventReleased:
			c.disarmHold(b)

		case b.Trigger == config.TriggerValue && ev.Kind == EventValue:
			c.dispatchValue(b, ev.Value)
		}
	}
}

func (c *Conductor) dispatchPress(b config.Binding, src Source) {
	var act Action
	var ok bool
	switch b.Mode {
	case config.ModeToggle:
		key := b.Key()
		c.mu.Lock()
		on := !c.toggles[key]
		c.toggles[key] = on
		c.mu.Unlock()
		act, ok = pressAction(b)
		if !on {
			act, ok = releaseAction(b)
		}
		c.updateFeedback(src, b, on)
	default: // momentary
		act, ok = pressAction(b)
	}
	if !ok || act.Address == "" {
		return
	}
	c.send(b, act)
}

func (c *Conductor) dispatchRelease(b config.Binding) {
	if b.Mode != config.ModeMomentary {
		return // toggle: release does nothing
	}
	act, ok := releaseAction(b)
	if !ok || act.Address == "" {
		return // nothing configured for release — normal for buttons
	}
	c.send(b, act)
}

func (c *Conductor) dispatchValue(b config.Binding, v float64) {
	act, ok := valueAction(b, v)
	if !ok {
		return
	}
	c.send(b, act)
}

func (c *Conductor) armHold(b config.Binding) {
	key := b.Key()
	d := time.Duration(b.HoldMs) * time.Millisecond
	if d <= 0 {
		d = 500 * time.Millisecond
	}
	c.mu.Lock()
	if t, exists := c.holdTimers[key]; exists {
		t.Stop()
	}
	c.holdTimers[key] = time.AfterFunc(d, func() {
		c.mu.Lock()
		delete(c.holdTimers, key)
		c.mu.Unlock()
		if act, ok := pressAction(b); ok && act.Address != "" {
			c.send(b, act)
		}
	})
	c.mu.Unlock()
}

func (c *Conductor) disarmHold(b config.Binding) {
	key := b.Key()
	c.mu.Lock()
	if t, exists := c.holdTimers[key]; exists {
		t.Stop()
		delete(c.holdTimers, key)
	}
	c.mu.Unlock()
	// Hold behaves like a long press; on release after a successful hold we
	// send the release action if one is configured (e.g. "stop" commands).
	act, ok := releaseAction(b)
	if ok && act.Address != "" {
		c.send(b, act)
	}
}

func (c *Conductor) updateFeedback(src Source, b config.Binding, on bool) {
	fb, ok := src.(FeedbackSink)
	if !ok {
		return
	}
	req := ControlFeedback{LED: &LEDState{Mode: "off"}}
	if on {
		req = ControlFeedback{LED: &LEDState{Mode: b.LEDMode(), Color: b.LEDColor()}}
	}
	if err := fb.SetControlFeedback(b.Control, req); err != nil {
		slog.Debug("SetControlFeedback failed", "source", src.ID(), "control", b.Control, "err", err)
	}
}

func (c *Conductor) send(b config.Binding, act Action) {
	c.mu.Lock()
	tgt, ok := c.targets[b.Target]
	c.mu.Unlock()
	if !ok {
		slog.Warn("binding references missing target", "binding", b.Key(), "target", b.Target)
		return
	}

	result := map[string]any{"binding": b.Key(), "ok": true, "action": act}
	if err := tgt.Send(act); err != nil {
		result["ok"] = false
		result["error"] = err.Error()
		slog.Warn("send failed", "target", tgt.ID(), "err", err)
	}
	c.sink.Broadcast("target.action", result)
}

func (c *Conductor) shutdownInstances() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.shutdownInstancesLocked()
}

func (c *Conductor) shutdownInstancesLocked() {
	if c.cancel != nil {
		c.cancel()
	}
	for key, t := range c.holdTimers {
		t.Stop()
		delete(c.holdTimers, key)
	}
	c.wg.Wait()
	for _, s := range c.sources {
		_ = s.Close()
	}
	for _, t := range c.targets {
		_ = t.Close()
	}
	// fresh context for next rebuild (Reload)
	parent := c.parentCtx
	if parent == nil {
		parent = context.Background()
	}
	c.ctx, c.cancel = context.WithCancel(parent)
}
