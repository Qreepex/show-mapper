package midi

import "fmt"

// Describe renders a raw MIDI message as a compact, human-readable line —
// used by `show-mapper midi monitor` and log output. Pure Go, no CGO needed.
func Describe(data []byte) string {
	if len(data) == 0 {
		return "<empty>"
	}
	status := data[0]
	typ := status & 0xF0
	ch := status & 0x0F
	switch typ {
	case 0x90:
		if len(data) >= 3 {
			if data[2] == 0 {
				return fmt.Sprintf("NoteOff      ch=%2d note=%3d  (raw % X)", ch, data[1], data)
			}
			return fmt.Sprintf("NoteOn       ch=%2d note=%3d vel=%3d  (raw % X)", ch, data[1], data[2], data)
		}
	case 0x80:
		if len(data) >= 3 {
			return fmt.Sprintf("NoteOff      ch=%2d note=%3d  (raw % X)", ch, data[1], data)
		}
	case 0xB0:
		if len(data) >= 3 {
			return fmt.Sprintf("ControlChange ch=%2d cc=%3d   val=%3d  (raw % X)", ch, data[1], data[2], data)
		}
	case 0xF0:
		return fmt.Sprintf("SysEx        len=%d  (raw % X)", len(data), data)
	}
	return fmt.Sprintf("MIDI         status=0x%02X  (raw % X)", status, data)
}
