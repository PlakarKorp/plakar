package signify

import (
	"os"
	"path/filepath"
	"testing"
)

// Fixtures were produced by the real toolchain — keys from gosignify -G,
// signatures from sumsign — so these tests fail if plakar ever stops agreeing
// with the tools that actually sign releases.
func fixture(t *testing.T, name string) []byte {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("reading fixture %s: %v", name, err)
	}

	return data
}
