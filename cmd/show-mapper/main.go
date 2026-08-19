// show-mapper — bridge physical control surfaces (MIDI boards, soon
// Stream Decks, ...) to show-software targets (grandMA3 via OSC, soon
// ArtNet/sACN, timecode).
//
// Subcommands:
//
//	show-mapper serve              run the backend + web UI (default)
//	show-mapper config init [path] write an annotated example config
//	show-mapper midi list          list OS MIDI in/out ports
//	show-mapper midi monitor <dev> dump decoded MIDI events (mapping helper)
//	show-mapper version            print build info
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Qreepex/show-mapper/internal/config"
	"github.com/Qreepex/show-mapper/internal/core"
	"github.com/Qreepex/show-mapper/internal/server"
	"github.com/Qreepex/show-mapper/internal/version"

	// Connectors register themselves via init(). Add future ones here
	// (internal/sources/streamdeck, internal/targets/artnet, ...).
	"github.com/Qreepex/show-mapper/internal/sources/midi"
	_ "github.com/Qreepex/show-mapper/internal/sources/sim"
	_ "github.com/Qreepex/show-mapper/internal/targets/osc"

	// Helper modules (action presets etc.) — each is self-contained with its
	// own docs and can be removed from this import list without core changes.
	_ "github.com/Qreepex/show-mapper/internal/helpers/gma3"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		switch args[0] {
		case "serve":
			return cmdServe(args[1:])
		case "config":
			return cmdConfig(args[1:])
		case "midi":
			return cmdMIDI(args[1:])
		case "version":
			fmt.Printf("show-mapper %s (commit %s, built %s)\n", version.Version, version.Commit, version.Date)
			return 0
		case "help":
			usage()
			return 0
		default:
			fmt.Fprintf(os.Stderr, "unknown subcommand %q\n\n", args[0])
			usage()
			return 2
		}
	}
	// no subcommand (or only flags) => serve
	return cmdServe(args)
}

func usage() {
	fmt.Fprintf(os.Stderr, `show-mapper %s — control-surface to show-software bridge

Usage:
  show-mapper [serve] [flags]    run backend + web UI (default action)
  show-mapper config init [path] write annotated example config (default ./show-mapper.yaml)
  show-mapper midi list          list OS MIDI in/out ports
  show-mapper midi monitor <dev> dump decoded MIDI events for a device (name substring)
  show-mapper version            print build info

serve flags:
  -config <path>   use a specific config file
  -listen <addr>   override http.listen for this run (e.g. 0.0.0.0:8484)

Docs: see docs/ (start with docs/architecture.md).
`, version.Version)
}

// --------------------------------------------------------------------------
// serve
// --------------------------------------------------------------------------

func cmdServe(args []string) int {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	cfgPath := fs.String("config", "", "path to config file (default: search order, see docs)")
	listen := fs.String("listen", "", "override http.listen (e.g. 0.0.0.0:8484)")
	noBrowser := fs.Bool("no-browser", false, "do not open the web UI in a browser at startup")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	var cfgFile *config.File
	var err error
	if *cfgPath != "" {
		cfgFile, err = config.LoadPath(*cfgPath)
		if err != nil && os.IsNotExist(err) {
			slog.Info("creating starter config at requested path", "path", *cfgPath)
			cfgFile, err = config.CreateDefault(*cfgPath)
		}
	} else {
		cfgFile, err = config.Load()
		if errors.Is(err, config.ErrNotFound) {
			p := config.PreferredCreatePath()
			slog.Info("no config found — generating starter config", "path", p)
			slog.Info("open the web UI (default http://127.0.0.1:8484) to finish setup")
			cfgFile, err = config.CreateDefault(p)
		}
	}
	if err != nil {
		slog.Error("loading config", "err", err)
		return 2
	}
	cfg := cfgFile.Get()
	if *listen != "" {
		cfg.HTTP.Listen = *listen
		cfgFile.Set(cfg)
	}
	feFalse := false
	if *noBrowser {
		cfg.HTTP.OpenBrowser = &feFalse
		cfgFile.Set(cfg)
	}
	slog.Info("config loaded", "path", cfgFile.Path,
		"sources", len(cfg.Sources), "targets", len(cfg.Targets),
		"bindings", len(cfg.Bindings))

	hub := server.NewHub()
	cond := core.NewConductor(cfgFile.Get(), hub)
	srv := server.New(cfgFile, cond, hub)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go cond.Run(ctx)

	// Optional startup update check (updates.autoCheck).
	if u := cfg.Updates; u != nil && u.AutoCheck && u.Repo != "" {
		go func() {
			st := srv.CheckForUpdate()
			switch {
			case st.Available:
				slog.Info("update available", "current", st.Current, "latest", st.LatestVersion, "url", st.LatestURL)
			case st.Error != "":
				slog.Debug("update check failed", "err", st.Error)
			}
		}()
	}

	// Hot-reload on hand edits of the config file (docs/architecture.md §4).
	go watchConfig(ctx, cfgFile.Path, func() {
		fresh, err := config.LoadPath(cfgFile.Path)
		if err != nil {
			slog.Warn("config file changed but failed to load", "err", err)
			return
		}
		if config.Equal(cfgFile.Get(), fresh.Get()) {
			return // our own Save() — already applied
		}
		if err := cond.Reload(fresh.Get()); err != nil {
			slog.Warn("config reload failed", "err", err)
			return
		}
		cfgFile.Set(fresh.Get())
		slog.Info("config reloaded from disk", "path", cfgFile.Path)
		hub.Broadcast("config.updated", fresh.Get())
	})

	if err := srv.Run(ctx); err != nil {
		slog.Error("server stopped with error", "err", err)
		return 1
	}
	slog.Info("bye")
	return 0
}

// --------------------------------------------------------------------------
// config
// --------------------------------------------------------------------------

func cmdConfig(args []string) int {
	if len(args) == 0 || args[0] != "init" {
		fmt.Fprintln(os.Stderr, "usage: show-mapper config init [path]")
		return 2
	}
	path := "./show-mapper.yaml"
	if len(args) > 1 {
		path = args[1]
	}
	if _, err := os.Stat(path); err == nil {
		fmt.Fprintf(os.Stderr, "refusing to overwrite existing %s\n", path)
		return 1
	}
	if err := os.WriteFile(path, []byte(exampleConfig), 0o600); err != nil {
		fmt.Fprintf(os.Stderr, "writing %s: %v\n", path, err)
		return 1
	}
	fmt.Printf("wrote %s\n\nNext steps:\n  1. edit %s (set your grandMA3 IP, pick your board)\n  2. show-mapper serve\n  3. open the web UI shown in the log (default http://127.0.0.1:8484)\n", path, path)
	return 0
}

// --------------------------------------------------------------------------
// midi helpers (mapping/discovery tools — see docs/midi-devices.md)
// --------------------------------------------------------------------------

func cmdMIDI(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: show-mapper midi <list|monitor>")
		return 2
	}
	switch args[0] {
	case "list":
		return midiList()
	case "monitor":
		return midiMonitor(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown midi subcommand %q (want list|monitor)\n", args[0])
		return 2
	}
}

func midiList() int {
	res, err := midi.InspectPorts()
	if err != nil {
		fmt.Fprintf(os.Stderr, "MIDI unavailable: %v\n", err)
		return 1
	}
	ports := res.(midi.PortList)
	fmt.Println("MIDI inputs:")
	for _, p := range ports.In {
		fmt.Printf("  [%d] %s\n", p.Number, p.Name)
	}
	fmt.Println("MIDI outputs:")
	for _, p := range ports.Out {
		fmt.Printf("  [%d] %s\n", p.Number, p.Name)
	}
	fmt.Println("\nUse a unique substring of the port name as `device` in your source config.")
	return 0
}

func midiMonitor(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: show-mapper midi monitor <device-name-substring>")
		return 2
	}
	match := args[0]

	hw, err := midi.NewHW()
	if err != nil {
		fmt.Fprintf(os.Stderr, "MIDI unavailable: %v\n", err)
		return 1
	}
	fmt.Printf("Monitoring MIDI ports matching %q — press buttons, move faders. Ctrl+C to stop.\n\n", match)
	conn, err := hw.Open(match, func(data []byte) {
		fmt.Printf("%-13s %s\n", time.Now().Format("15:04:05.000"), midi.Describe(data))
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	defer func() { _ = conn.Close() }()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()
	return 0
}
