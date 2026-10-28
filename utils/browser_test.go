//go:build !windows

package utils

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// stubLauncher puts a fake open/xdg-open first in PATH. It records its
// arguments in the returned file.
func stubLauncher(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	out := filepath.Join(dir, "args")
	name := "xdg-open"
	if runtime.GOOS == "darwin" {
		name = "open"
	}
	script := "#!/bin/sh\necho \"$@\" > " + out + "\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(script), 0755))
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return out
}

func TestBrowserTrySpawnOpensHTTPURLs(t *testing.T) {
	for _, u := range []string{"https://plakar.io/login?code=1", "http://127.0.0.1:8080/"} {
		t.Run(u, func(t *testing.T) {
			out := stubLauncher(t)
			require.NoError(t, BrowserTrySpawn(u))
			require.Eventually(t, func() bool {
				data, err := os.ReadFile(out)
				return err == nil && strings.TrimSpace(string(data)) == u
			}, 3*time.Second, 10*time.Millisecond)
		})
	}
}

func TestBrowserTrySpawnRefusesOtherURLs(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"empty", ""},
		{"leading dash", "-x"},
		{"option", "--help"},
		{"file scheme", "file:///etc/passwd"},
		{"javascript scheme", "javascript:alert(1)"},
		{"ftp scheme", "ftp://example.com/"},
		{"no scheme", "example.com/login"},
		{"no host", "https://"},
		{"leading space", " https://example.com/"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := stubLauncher(t)
			require.Error(t, BrowserTrySpawn(tt.url))
			_, err := os.Stat(out)
			require.ErrorIs(t, err, os.ErrNotExist, "launcher must not run")
		})
	}
}
