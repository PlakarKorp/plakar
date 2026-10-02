//go:build !windows && !darwin

package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNoGraphicalSession(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		wantErr bool
	}{
		{"no display", nil, true},
		{"x11", map[string]string{"DISPLAY": ":0"}, false},
		{"wayland", map[string]string{"WAYLAND_DISPLAY": "wayland-0"}, false},
		{"wsl", map[string]string{"WSL_DISTRO_NAME": "Ubuntu"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := noGraphicalSession(func(k string) string { return tt.env[k] })
			if tt.wantErr {
				require.ErrorIs(t, err, errNoGraphicalSession)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestBrowserTrySpawnHeadless(t *testing.T) {
	t.Setenv("DISPLAY", "")
	t.Setenv("WAYLAND_DISPLAY", "")
	t.Setenv("WSL_DISTRO_NAME", "")

	require.ErrorIs(t, BrowserTrySpawn("https://example.com"), errNoGraphicalSession)
}
