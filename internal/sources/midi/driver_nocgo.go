//go:build !cgo

package midi

// NewHW fails permanently in non-CGO builds: no OS MIDI stack is available.
// This keeps `go build`, `go test` and frontend development possible
// everywhere; release binaries are always built with CGO_ENABLED=1.
func NewHW() (HW, error) { return nil, ErrNoCGO }
