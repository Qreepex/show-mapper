package server

import (
	"embed"
	"io/fs"
)

// dist holds the compiled Svelte frontend. The web build
// (web/ — SvelteKit + adapter-static) writes directly into this directory;
// a placeholder index.html is committed so plain `go build` works on a
// fresh clone (CI/release builds always run the web build first).
//
//go:embed all:dist
var distEmbed embed.FS

func frontendFS() (fs.FS, error) {
	return fs.Sub(distEmbed, "dist")
}
