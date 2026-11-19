package cmd

import (
	"runtime/debug"
	"testing"
)

func TestResolveVersion(t *testing.T) {
	info := func(v string) *debug.BuildInfo { return &debug.BuildInfo{Main: debug.Module{Version: v}} }
	tests := []struct {
		name     string
		injected string
		info     *debug.BuildInfo
		want     string
	}{
		{"injected wins", "0.4.0", info("v0.3.0"), "0.4.0"},
		{"go install", "dev", info("v0.4.0"), "v0.4.0"},
		{"empty injected", "", info("v0.4.0"), "v0.4.0"},
		{"local build", "dev", info("(devel)"), "dev"},
		{"empty module version", "dev", info(""), "dev"},
		{"no build info", "dev", nil, "dev"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ResolveVersion(tt.injected, tt.info); got != tt.want {
				t.Errorf("ResolveVersion(%q) = %q, want %q", tt.injected, got, tt.want)
			}
		})
	}
}
