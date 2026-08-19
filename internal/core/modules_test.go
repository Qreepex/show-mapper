package core

import (
	"context"
	"testing"
	"time"

	"github.com/Qreepex/show-mapper/internal/config"
)

// The app must be complete WITHOUT any modules: no sources/targets/binding,
// and the conductor must idle happily (UI, REST, WS all still work).
func TestConductorRunsWithZeroModules(t *testing.T) {
	cond := NewConductor(config.Config{Version: 1, HTTP: config.HTTPConfig{Listen: "127.0.0.1:0"}}, NopSink{})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { cond.Run(ctx); close(done) }()

	time.Sleep(50 * time.Millisecond)
	if got := cond.Snapshot(); len(got) != 0 {
		t.Fatalf("expected empty snapshot, got %+v", got)
	}

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("conductor did not stop")
	}
}

// Unknown module types (e.g. config references a module not compiled in —
// someone removed the import) must not crash or block everything else:
// the bad instance is skipped with an error log; valid neighbors keep running.
func TestConductorSurvivesUnknownModuleTypes(t *testing.T) {
	cfg := config.Config{
		Version: 1,
		HTTP:    config.HTTPConfig{Listen: "127.0.0.1:0"},
		Sources: []config.SourceConfig{
			{ID: "ghost", Type: "not-a-real-type"},
			{ID: "wing", Type: "fake"}, // registered in binding_test.go init()
		},
		Targets: []config.TargetConfig{
			{ID: "gone", Type: "also-fake"},
			{ID: "out", Type: "fake"},
		},
	}
	cond := NewConductor(cfg, NopSink{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go cond.Run(ctx)

	// the valid fake pair still connects and dispatches
	src, err := waitFor(func() (*fakeSource, bool) { return testSrc, testSrc != nil })
	if err != nil {
		t.Fatal("valid source not started despite unknown sibling:", err)
	}
	src.events <- Event{SourceID: "wing", Control: "pad-0-0", Kind: EventPressed, Value: 1, When: time.Now()}
	// (no binding references it — just asserting the pump is alive; the dispatch
	// path is covered in TestConductorDispatchMomentary)
	time.Sleep(20 * time.Millisecond)

	snap := cond.Snapshot()
	if len(snap) != 2 { // only the two "fake" instances
		t.Fatalf("unknown types must be skipped, snapshot = %+v", snap)
	}
	for _, c := range snap {
		if c.Type != "fake" {
			t.Errorf("unexpected connector in snapshot: %+v", c)
		}
	}
}
