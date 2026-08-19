// Package server exposes the embedded web UI, the JSON REST API and the
// realtime WebSocket stream. All endpoint shapes are documented in
// docs/protocols.md.
package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/yourorg/showbridge/internal/config"
	"github.com/yourorg/showbridge/internal/core"
	"github.com/yourorg/showbridge/internal/version"
)

// Server wires HTTP routes to the config store and the conductor.
type Server struct {
	cfgFile *config.File
	cond    *core.Conductor
	hub     *Hub
	http    *http.Server
}

// New builds the server. hub receives broadcasts AND serves /ws.
func New(cfgFile *config.File, cond *core.Conductor, hub *Hub) *Server {
	s := &Server{cfgFile: cfgFile, cond: cond, hub: hub}
	hub.SetSnapshot(s.snapshot)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("GET /api/meta", s.handleMeta)
	mux.HandleFunc("GET /api/config", s.handleGetConfig)
	mux.HandleFunc("PUT /api/config", s.handlePutConfig)
	mux.HandleFunc("GET /api/state", s.handleState)
	mux.HandleFunc("GET /api/sources/{type}/inspect", s.handleInspect)
	mux.HandleFunc("GET /ws", hub.ServeWS)
	mux.HandleFunc("GET /", s.handleFrontend)

	s.http = &http.Server{
		Addr:              cfgFile.Config.HTTP.Listen,
		Handler:           logRequests(mux),
		ReadHeaderTimeout: 10 * time.Second,
	}
	return s
}

// Run starts serving and blocks until ctx is done (graceful shutdown).
func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		slog.Info("web UI listening", "addr", "http://"+s.http.Addr)
		errCh <- s.http.ListenAndServe()
	}()
	select {
	case <-ctx.Done():
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.http.Shutdown(shutCtx)
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

// snapshot is the first WS message a client gets after connect.
func (s *Server) snapshot() Envelope {
	return Envelope{
		Type: "state.snapshot",
		TS:   time.Now(),
		Data: map[string]any{
			"version":    version.Version,
			"commit":     version.Commit,
			"connectors": s.cond.Snapshot(),
			"config":     s.cfgFile.Config,
		},
	}
}

// ---------------------------------------------------------------------------
// REST handlers
// ---------------------------------------------------------------------------

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"ok": true, "version": version.Version, "clients": s.hub.ClientCount(),
	})
}

func (s *Server) handleMeta(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"version":        version.Version,
		"commit":         version.Commit,
		"sourceTypes":    core.SourceTypeInfos(),
		"targetTypes":    core.TargetTypeInfos(),
		"triggers":       config.Triggers,
		"modes":          config.Modes,
		"actionTypes":    config.ActionTypes,
		"controlKinds":   config.ControlKinds,
		"ledStyles":      config.LEDStyles,
		"customProfiles": filterCustomProfiles(s.cfgFile.Config.Profiles),
	})
}

// filterCustomProfiles groups user-defined profiles by connector type so the
// UI can merge them with the built-in ones from SourceTypeInfos.
func filterCustomProfiles(list []config.ProfileConfig) map[string][]config.ProfileConfig {
	out := map[string][]config.ProfileConfig{}
	for _, p := range list {
		out[p.Type] = append(out[p.Type], p)
	}
	return out
}

func (s *Server) handleGetConfig(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.cfgFile.Config)
}

func (s *Server) handlePutConfig(w http.ResponseWriter, r *http.Request) {
	var cfg config.Config
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cfg); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid JSON: %w", err))
		return
	}
	if err := config.Normalize(&cfg); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	s.cfgFile.Config = cfg
	if err := s.cfgFile.Save(); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("saving: %w", err))
		return
	}
	if err := s.cond.Reload(cfg); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	s.hub.Broadcast("config.updated", cfg)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleState(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"connectors": s.cond.Snapshot(),
		"configPath": s.cfgFile.Path,
	})
}

// handleInspect runs the inspector of a source type (e.g. list MIDI ports).
func (s *Server) handleInspect(w http.ResponseWriter, r *http.Request) {
	typ := r.PathValue("type")
	if !containsStr(core.SourceTypes(), typ) {
		writeError(w, http.StatusNotFound, fmt.Errorf("unknown source type %q", typ))
		return
	}
	res, err := core.InspectSourceType(typ)
	if err != nil {
		// e.g. MIDI in a CGO_ENABLED=0 build — a 200 with ok:false keeps the
		// UI able to render a helpful message instead of a hard error.
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	if res == nil {
		writeError(w, http.StatusNotFound, fmt.Errorf("source type %q has no inspector", typ))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "result": res})
}

// ---------------------------------------------------------------------------
// Frontend (embedded SPA with history-fallback)
// ---------------------------------------------------------------------------

func (s *Server) handleFrontend(w http.ResponseWriter, r *http.Request) {
	fsys, err := frontendFS()
	if err != nil {
		http.Error(w, "frontend not embedded", http.StatusInternalServerError)
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		path = "index.html"
	}
	info, err := fs.Stat(fsys, path)
	if err != nil || info.IsDir() {
		// SPA fallback (SvelteKit adapter-static emits 200.html for this).
		// 404 for missing asset-ish paths instead of serving HTML for typos.
		if strings.Contains(path, ".") && !strings.HasSuffix(path, ".html") {
			http.NotFound(w, r)
			return
		}
		path = "200.html"
	}
	http.ServeFileFS(w, r, fsys, path)
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, err error) {
	// errors.Join output is newline-separated; surface as list for the UI.
	parts := strings.Split(err.Error(), "\n")
	writeJSON(w, status, map[string]any{"ok": false, "errors": parts})
}

func containsStr(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api") {
			slog.Debug("http", "method", r.Method, "path", r.URL.Path)
		}
		next.ServeHTTP(w, r)
	})
}
