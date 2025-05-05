package snapshot_test

import (
	"strings"
	"testing"

	"github.com/PlakarKorp/plakar/snapshot"
	_ "github.com/PlakarKorp/plakar/snapshot/exporter/fs"
	"github.com/stretchr/testify/require"
)

func TestCheck(t *testing.T) {
	_, snap := generateSnapshot(t)
	defer snap.Close()

	// search for the correct filepath as the path was mkdir temp we cannot hardcode it
	var filepath string
	fs, err := snap.Filesystem()
	require.NoError(t, err)
	for pathname, err := range fs.Pathnames() {
		require.NoError(t, err)
		if strings.Contains(pathname, "dummy.txt") {
			filepath = pathname
		}
	}
	require.NotEmpty(t, filepath)

	err = snap.Check(filepath, &snapshot.CheckOptions{
		MaxConcurrency: 1,
	})
	require.NoError(t, err)
}
