package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	ignoreFileName          = "ignores"
	ignoreFileContent       = "# header\n\npat1\npat2\n  \t\n  \t# leading-space comment is NOT stripped\n"
	ignoreFileMode          = 0o600
	missingIgnoreFilePath   = "/no/such/file"
	unableToOpenErrorMarker = "unable to open"
)

var expectedIgnoreFileLines = []string{"pat1", "pat2", "  \t# leading-space comment is NOT stripped"}

func TestLoadIgnoreFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), ignoreFileName)
	require.NoError(t, os.WriteFile(path, []byte(ignoreFileContent), ignoreFileMode))

	lines, err := LoadIgnoreFile(path)
	require.NoError(t, err)
	require.Equal(t, expectedIgnoreFileLines, lines)
}

func TestLoadIgnoreFileMissing(t *testing.T) {
	_, err := LoadIgnoreFile(missingIgnoreFilePath)
	require.Error(t, err)
	require.Contains(t, err.Error(), unableToOpenErrorMarker)
}

func TestSourceIgnoreRules(t *testing.T) {
	path := filepath.Join(t.TempDir(), ignoreFileName)
	require.NoError(t, os.WriteFile(path, []byte("# header\n\nfrom-file\n"), ignoreFileMode))

	for _, tc := range []struct {
		name   string
		source map[string]string
		want   []string
	}{
		{
			name:   "no rules",
			source: map[string]string{"location": "fs:/tmp"},
		},
		{
			name:   "comma separated list is trimmed",
			source: map[string]string{"ignore": " *.tmp , , *.log "},
			want:   []string{"*.tmp", "*.log"},
		},
		{
			name:   "ignore file",
			source: map[string]string{"ignore-file": path},
			want:   []string{"from-file"},
		},
		{
			name:   "file rules come before inline ones",
			source: map[string]string{"ignore-file": path, "ignore": "*.tmp"},
			want:   []string{"from-file", "*.tmp"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rules, err := SourceIgnoreRules(tc.source)
			require.NoError(t, err)
			require.Equal(t, tc.want, rules)
		})
	}
}

func TestSourceIgnoreRulesMissingFile(t *testing.T) {
	_, err := SourceIgnoreRules(map[string]string{"ignore-file": missingIgnoreFilePath})
	require.Error(t, err)
	require.Contains(t, err.Error(), unableToOpenErrorMarker)
}
