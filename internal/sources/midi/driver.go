// Package midi implements the USB-MIDI source connector: it maps physical
// controls of class-compliant MIDI controllers (APC mini / mini mk2, ...)
// to core.Events using a device Profile, and provides LED feedback.
//
// Hardware access is abstracted behind the HW interface, which has two
// implementations selected by build constraints:
//
//   - driver_cgo.go   (requires CGO, uses RtMidi — works on Win/macOS/Linux)
//   - driver_nocgo.go (stub: returns ErrNoCGO so non-CGO builds still
//     compile, test and run the web UI; MIDI just reports an error state)
//
// See docs/midi-devices.md for why no vendor drivers are needed and how to
// add a new device profile.
package midi

import "errors"

// ErrNoCGO is returned by NewHW in builds without CGO support.
var ErrNoCGO = errors.New("no MIDI hardware support in this build: it was compiled with CGO_ENABLED=0 — download a release binary or build with CGO_ENABLED=1 (see docs/releasing.md)")

// PortInfo describes an available MIDI port.
type PortInfo struct {
	Number int    `json:"number"`
	Name   string `json:"name"`
}

// HW abstracts the OS MIDI stack (WinMM/CoreMIDI/ALSA via RtMidi).
type HW interface {
	// InPorts lists MIDI input ports.
	InPorts() ([]PortInfo, error)
	// OutPorts lists MIDI output ports.
	OutPorts() ([]PortInfo, error)
	// Open opens the first input and (best effort) matching output port whose
	// name contains match (case-insensitive). Output is optional: buttons and
	// LEDs work without it, LED feedback just stays dark.
	// onMessage is called for every inbound raw message (status byte + data).
	Open(match string, onMessage func(data []byte)) (Conn, error)
}

// Conn is an open device connection.
type Conn interface {
	// Send transmits raw bytes (e.g. {0x90|ch, note, vel}) to the output port.
	// Returns an error if no output port is open.
	Send(data []byte) error
	Close() error
	InPortName() string
	OutPortName() string // "" if no output open
}
