package main

import (
	"context"
	"log/slog"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// watchConfig reloads the config file when it changes on disk (hand-edits).
// Saves made through the app itself are de-duplicated by the callback
// (sameConfig) and coalesced by a debounce window.
func watchConfig(ctx context.Context, path string, onChange func()) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		slog.Warn("config watcher unavailable", "err", err)
		return
	}
	defer func() { _ = watcher.Close() }()

	if err := watcher.Add(filepath.Dir(path)); err != nil {
		slog.Warn("config watcher cannot watch file", "path", path, "err", err)
		return
	}
	base := filepath.Clean(path)

	var mu sync.Mutex
	var timer *time.Timer
	debounce := func() {
		mu.Lock()
		defer mu.Unlock()
		if timer != nil {
			timer.Stop()
		}
		timer = time.AfterFunc(300*time.Millisecond, onChange)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-watcher.Events:
			if !ok {
				return
			}
			if filepath.Clean(ev.Name) != base {
				continue
			}
			if ev.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) != 0 {
				debounce()
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			slog.Debug("config watcher error", "err", err)
		}
	}
}
