// Browser-opening helpers (no extra dependency; best effort, never fatal).
package server

import (
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"strings"
)

// openUIURL converts a listen address into the URL a browser should open.
// Wildcard binds (0.0.0.0 / "") map to loopback for browser use.
func openUIURL(listen string) string {
	host, port, err := net.SplitHostPort(listen)
	if err != nil {
		return "http://" + listen // best effort
	}
	switch host {
	case "", "0.0.0.0", "::", "::1":
		host = "127.0.0.1"
	}
	if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	return fmt.Sprintf("http://%s:%s", host, port)
}

// OpenBrowser opens url in the system default browser (best effort).
// On WSL launches are delegated through Windows-side tools.
func OpenBrowser(url string) error {
	if runtime.GOOS == "windows" {
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	}
	if runtime.GOOS == "darwin" {
		return exec.Command("open", url).Start()
	}
	// Linux/bsd: handle WSL separately (xdg-open may point at a text browser or nothing).
	if isWSL() {
		for _, c := range [][]string{
			{"wslview", url},
			{"powershell.exe", "-NoProfile", "-Command", "Start-Process", url},
			{"cmd.exe", "/c", "start", url},
			{"xdg-open", url},
		} {
			if err := tryRun(c[0], c[1:]...); err == nil {
				return nil
			}
		}
		return fmt.Errorf("no WSL browser launcher found (wslview/powershell.exe/cmd.exe/xdg-open)")
	}
	for _, c := range [][]string{{"xdg-open", url}, {"gio", "open", url}} {
		if err := tryRun(c[0], c[1:]...); err == nil {
			return nil
		}
	}
	return fmt.Errorf("no browser launcher found (xdg-open)")
}

func tryRun(name string, args ...string) error {
	if _, err := exec.LookPath(name); err != nil {
		return err
	}
	return exec.Command(name, args...).Start()
}
