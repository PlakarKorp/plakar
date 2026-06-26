package mount

import (
	"bytes"
	"testing"

	"github.com/PlakarKorp/plakar/subcommands"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

// NOTE: Execute() is intentionally not covered here. It ends in
// fuse.ExecuteFUSE or http.ExecuteHTTP; the FUSE path requires a real OS mount
// (privileges + platform support) unavailable in the test environment. Only
// Parse is exercised. The http branch is covered by subcommands/mount/http.

func TestMountRegistered(t *testing.T) {
	cmd, matched, _ := subcommands.Lookup([]string{"mount", "-to", "/mnt"})
	require.NotNil(t, cmd)
	require.Equal(t, []string{"mount"}, matched)
	_, ok := cmd.(*Mount)
	require.True(t, ok)
}

func TestParseRepositoryLevel(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	_, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)

	cmd := &Mount{}
	err := cmd.Parse(ctx, []string{"-to", "http://localhost:9999"})
	require.NoError(t, err)
	require.Equal(t, "http://localhost:9999", cmd.Mountpoint)
	require.Empty(t, cmd.SnapshotPath)
	require.NotNil(t, cmd.LocateOptions)
}

func TestParseSnapshotLevelResetsLocateOptions(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	_, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)

	cmd := &Mount{}
	// A single positional arg switches to snapshot level.
	err := cmd.Parse(ctx, []string{"-to", "/mnt", "abcdef:/some/path"})
	require.NoError(t, err)
	require.Equal(t, "/mnt", cmd.Mountpoint)
	require.Equal(t, "abcdef:/some/path", cmd.SnapshotPath)
	require.NotNil(t, cmd.LocateOptions)
}

func TestParseNoArgs(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	_, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)

	cmd := &Mount{}
	err := cmd.Parse(ctx, nil)
	require.NoError(t, err)
	require.Empty(t, cmd.Mountpoint)
	require.Empty(t, cmd.SnapshotPath)
	require.NotNil(t, cmd.LocateOptions)
}
