package server

import "testing"

func TestOpenUIURL(t *testing.T) {
	cases := map[string]string{
		"127.0.0.1:8484":    "http://127.0.0.1:8484",
		"0.0.0.0:8484":      "http://127.0.0.1:8484",
		":8484":             "http://127.0.0.1:8484",
		"192.168.0.10:9000": "http://192.168.0.10:9000",
		"[::1]:8484":        "http://127.0.0.1:8484",
		"garbage":           "http://garbage",
	}
	for in, want := range cases {
		if got := openUIURL(in); got != want {
			t.Errorf("openUIURL(%q) = %q, want %q", in, got, want)
		}
	}
}
