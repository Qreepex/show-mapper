// showbridge — bridge physical control surfaces (MIDI boards, soon
// Stream Decks, ...) to show-software targets (grandMA3 via OSC, soon
// ArtNet/sACN, timecode).
//
// Subcommands:
//
//	showbridge serve              run the backend + web UI (default)
//	showbridge config init [path] write an annotated example config
//	showbridge midi list          list OS MIDI in/out ports
//	showbridge midi monitor <dev> dump decoded MIDI events (mapping helper)
//	showbridge version            print build info
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/yourorg/showbridge/internal/config"
	"github.com/yourorg/showbridge/internal/core"
	"github.com/yourorg/showbridge/internal/server"
	"github.com/yourorg/showbridge/internal/version"

	// Connectors register themselves via init(). Add future ones here
	// (internal/sources/streamdeck, internal/targets/artnet, ...).
	"github.com/yourorg/showbridge/internal/sources/midi"
	_ "github.com/yourorg/showbridge/internal/targets/osc"
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
			fmt.Printf("showbridge %s (commit %s, built %s)\n", version.Version, version.Commit, version.Date)
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
	fmt.Fprintf(os.Stderr, `showbridge %s — control-surface to show-software bridge

Usage:
  showbridge [serve] [flags]    run backend + web UI (default action)
  showbridge config init [path] write annotated example config (default ./showbridge.yaml)
  showbridge midi list          list OS MIDI in/out ports
  showbridge midi monitor <dev> dump decoded MIDI events for a device (name substring)
  showbridge version            print build info

serve flags:
  -config <path>   use a specific config file
  -listen <addr>   override http.listen for this run (e.g. 0.0.0.0:8080)

Docs: see docs/ (start with docs/architecture.md).
`, version.Version)
}

// --------------------------------------------------------------------------
// serve
// --------------------------------------------------------------------------

func cmdServe(args []string) int {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	cfgPath := fs.String("config", "", "path to config file (default: search order, see docs)")
	listen := fs.String("listen", "", "override http.listen (e.g. 0.0.0.0:8080)")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	var cfgFile *config.File
	var err error
	if *cfgPath != "" {
		cfgFile, err = config.LoadPath(*cfgPath)
	} else {
		cfgFile, err = config.Load()
	}
	if err != nil {
		slog.Error("loading config", "err", err)
		slog.Info("hint: run `showbridge config init` to create a starter config")
		return 2
	}
	if *listen != "" {
		cfgFile.Config.HTTP.Listen = *listen
	}
	slog.Info("config loaded", "path", cfgFile.Path,
		"sources", len(cfgFile.Config.Sources), "targets", len(cfgFile.Config.Targets),
		"bindings", len(cfgFile.Config.Bindings))

	hub := server.NewHub()
	cond := core.NewConductor(cfgFile.Config, hub)
	srv := server.New(cfgFile, cond, hub)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go cond.Run(ctx)

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
		fmt.Fprintln(os.Stderr, "usage: showbridge config init [path]")
		return 2
	}
	path := "./showbridge.yaml"
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
	fmt.Printf("wrote %s\n\nNext steps:\n  1. edit %s (set your grandMA3 IP, pick your board)\n  2. showbridge serve\n  3. open the web UI shown in the log (default http://127.0.0.1:8080)\n", path, path)
	return 0
}

// --------------------------------------------------------------------------
// midi helpers (mapping/discovery tools — see docs/midi-devices.md)
// --------------------------------------------------------------------------

func cmdMIDI(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: showbridge midi <list|monitor>")
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
		fmt.Fprintln(os.Stderr, "usage: showbridge midi monitor <device-name-substring>")
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
