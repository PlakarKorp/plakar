package mount

import (
	"bytes"
	"testing"

	_ "github.com/PlakarKorp/integrations/fs/exporter"
	"github.com/PlakarKorp/plakar/subcommands"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

func TestMountRegisteredFactory(t *testing.T) {
	cmd, _, _ := subcommands.Lookup([]string{"mount"})
	require.NotNil(t, cmd)
	require.IsType(t, &Mount{}, cmd)
}

func TestMountParseRepositoryLevel(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	_ = repo

	cmd := &Mount{}
	require.NoError(t, cmd.Parse(ctx, []string{"-to", "/mnt/x"}))
	require.Equal(t, "/mnt/x", cmd.Mountpoint)
	require.Equal(t, "", cmd.SnapshotPath)
	require.NotNil(t, cmd.LocateOptions)
}

func TestMountParseSnapshotLevel(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	_ = repo

	cmd := &Mount{}
	require.NoError(t, cmd.Parse(ctx, []string{"-to", "/mnt/x", "abc123"}))
	require.Equal(t, "abc123", cmd.SnapshotPath)
	require.True(t, cmd.LocateOptions.Empty(), "snapshot-level parse resets LocateOptions")
}

func TestMountParseAllowOthers(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	_ = repo

	cmd := &Mount{}
	require.NoError(t, cmd.Parse(ctx, []string{"-allow-others", "-to", "/mnt/x"}))
	require.True(t, cmd.AllowOthers)
}

func TestMountParseToken(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	_ = repo

	cmd := &Mount{}
	require.NoError(t, cmd.Parse(ctx, []string{"-token", "s3cret", "-to", "http://127.0.0.1:0"}))
	require.Equal(t, "s3cret", cmd.Token)
	require.False(t, cmd.NoAuth)

	cmd = &Mount{}
	require.NoError(t, cmd.Parse(ctx, []string{"-no-auth", "-to", "http://127.0.0.1:0"}))
	require.Equal(t, "", cmd.Token)
	require.True(t, cmd.NoAuth)
}

func TestMountExecuteBadSnapshot(t *testing.T) {
	// A snapshot path that resolves to nothing fails before any mount is
	// attempted (so this is safe on all platforms, FUSE never engages).
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockFile("a.txt", 0644, "x"),
	})
	defer snap.Close()

	cmd := &Mount{}
	require.NoError(t, cmd.Parse(ctx, []string{"-to", "/mnt/x", "deadbeefdeadbeef:/"}))
	status, err := cmd.Execute(ctx, repo)
	require.Error(t, err)
	require.Equal(t, 1, status)
}
