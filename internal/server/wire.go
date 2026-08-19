package server

import (
	"time"

	"github.com/Qreepex/show-mapper/internal/config"
	"github.com/Qreepex/show-mapper/internal/core"
)

// Wire-format types shared with the frontend. This file is the only part of
// the server package fed into tygo (see tygo.yaml exclude_files): change
// these structs, then run `make types` to regenerate web/src/lib/generated/.

// Envelope is the single wire format for all WebSocket traffic.
// Message types are dot-namespaced, see docs/protocols.md.
type Envelope struct {
	Type string    `json:"type"`
	TS   time.Time `json:"ts"`
	Data any       `json:"data"`
}

// Snapshot is the payload of the initial "state.snapshot" WS message.
type Snapshot struct {
	Version    string              `json:"version"`
	Commit     string              `json:"commit"`
	Connectors []core.SnapshotConn `json:"connectors"`
	Config     config.Config       `json:"config"`
}
