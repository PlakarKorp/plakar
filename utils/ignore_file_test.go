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
