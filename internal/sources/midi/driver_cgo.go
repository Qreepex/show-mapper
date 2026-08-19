//go:build cgo

package midi

import (
	"fmt"
	"strings"
	"sync"

	rtmidi "gitlab.com/gomidi/rtmididrv/imported/rtmidi"
)

// RtMidi is a cross-platform C++ wrapper (bundled with this dependency) around
// the OS MIDI stacks: Windows Multimedia (WinMM), CoreMIDI (macOS) and ALSA
// (Linux). Build requirements per OS are documented in docs/releasing.md
// (e.g. libasound2-dev on Linux, a C++ toolchain everywhere).

// rtHW is backed by RtMidi. Enumeration creates short-lived probe instances;
// Open() creates its own MIDIIn/MIDIOut pair for the matched device.
type rtHW struct{}

// NewHW returns the RtMidi-backed implementation.
func NewHW() (HW, error) { return rtHW{}, nil }

func (rtHW) InPorts() ([]PortInfo, error) {
	probe, err := rtmidi.NewMIDIInDefault()
	if err != nil {
		return nil, fmt.Errorf("midi probe: %w", err)
	}
	defer probe.Destroy()
	n, err := probe.PortCount()
	if err != nil {
		return nil, err
	}
	out := make([]PortInfo, 0, n)
	for i := 0; i < n; i++ {
		name, err := probe.PortName(i)
		if err != nil {
			continue
		}
		out = append(out, PortInfo{Number: i, Name: name})
	}
	return out, nil
}

func (rtHW) OutPorts() ([]PortInfo, error) {
	probe, err := rtmidi.NewMIDIOutDefault()
	if err != nil {
		return nil, fmt.Errorf("midi probe: %w", err)
	}
	defer probe.Destroy()
	n, err := probe.PortCount()
	if err != nil {
		return nil, err
	}
	out := make([]PortInfo, 0, n)
	for i := 0; i < n; i++ {
		name, err := probe.PortName(i)
		if err != nil {
			continue
		}
		out = append(out, PortInfo{Number: i, Name: name})
	}
	return out, nil
}

func (rtHW) Open(match string, onMessage func(data []byte)) (Conn, error) {
	if strings.TrimSpace(match) == "" {
		return nil, fmt.Errorf("device match string is empty")
	}
	inIdx, inName, err := findInPort(match)
	if err != nil {
		return nil, err
	}

	in, err := rtmidi.NewMIDIIn(rtmidi.APIUnspecified, "show-mapper", 1024)
	if err != nil {
		return nil, fmt.Errorf("creating MIDI input: %w", err)
	}
	// Keep sysex (device handshakes/responses use it), drop timing/sensing noise.
	if err := in.IgnoreTypes(false, true, true); err != nil {
		in.Destroy()
		return nil, err
	}
	if err := in.SetCallback(func(_ rtmidi.MIDIIn, msg []byte, _ float64) {
		onMessage(msg)
	}); err != nil {
		in.Destroy()
		return nil, err
	}
	if err := in.OpenPort(inIdx, "show-mapper"); err != nil {
		in.Destroy()
		return nil, fmt.Errorf("opening input %q: %w", inName, err)
	}

	c := &rtConn{in: in, inName: inName}

	// Output is optional (needed for LED feedback / sysex handshake).
	if outIdx, outName, err := findOutPort(match); err == nil {
		if out, err := rtmidi.NewMIDIOutDefault(); err == nil {
			if err := out.OpenPort(outIdx, "show-mapper"); err == nil {
				c.out = out
				c.outName = outName
			} else {
				out.Destroy()
			}
		}
	}
	return c, nil
}

func findInPort(match string) (int, string, error) {
	probe, err := rtmidi.NewMIDIInDefault()
	if err != nil {
		return 0, "", fmt.Errorf("midi probe: %w", err)
	}
	defer probe.Destroy()
	return findPort(probe, match, "input")
}

func findOutPort(match string) (int, string, error) {
	probe, err := rtmidi.NewMIDIOutDefault()
	if err != nil {
		return 0, "", fmt.Errorf("midi probe: %w", err)
	}
	defer probe.Destroy()
	return findPort(probe, match, "output")
}

// findPort locates a port index by case-insensitive name substring.
// Port indices are enumerations of the currently attached devices — a device
// plugged in later gets a new index; the conductor's retry loop re-enumerates.
func findPort(ports rtmidi.MIDI, match, dir string) (int, string, error) {
	n, err := ports.PortCount()
	if err != nil {
		return 0, "", err
	}
	m := strings.ToLower(match)
	var available []string
	for i := 0; i < n; i++ {
		name, err := ports.PortName(i)
		if err != nil {
			continue
		}
		available = append(available, name)
		if strings.Contains(strings.ToLower(name), m) {
			return i, name, nil
		}
	}
	return 0, "", fmt.Errorf("no MIDI %s port matching %q (available: %s)", dir, match, strings.Join(available, ", "))
}

type rtConn struct {
	mu      sync.Mutex
	in      rtmidi.MIDIIn
	out     rtmidi.MIDIOut
	inName  string
	outName string
}

func (c *rtConn) InPortName() string { return c.inName }

func (c *rtConn) OutPortName() string { return c.outName }

func (c *rtConn) Send(data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.out == nil {
		return fmt.Errorf("no MIDI output port open (LED feedback / sysex unavailable)")
	}
	return c.out.SendMessage(data)
}

func (c *rtConn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.in != nil {
		_ = c.in.CancelCallback()
		_ = c.in.Close()
		c.in.Destroy()
		c.in = nil
	}
	if c.out != nil {
		_ = c.out.Close()
		c.out.Destroy()
		c.out = nil
	}
	return nil
}
