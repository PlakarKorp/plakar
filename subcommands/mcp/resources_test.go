package mcp

import (
	"bytes"
	"encoding/hex"
	"testing"

	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

func TestReadResource(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	snap := ptesting.GenerateSnapshot(t, repo, mockFiles("hello dummy"))
	snap.Close()
	snapID := hex.EncodeToString(snap.Header.GetIndexShortID())

	data, _, err := readResource(repo, snapID, "/subdir/dummy.txt", 1<<20)
	require.NoError(t, err)
	require.Equal(t, "hello dummy", string(data))

	// unlike read_file, a resource cannot be paged: too large is an error
	_, _, err = readResource(repo, snapID, "/subdir/dummy.txt", 4)
	require.Error(t, err)

	_, _, err = readResource(repo, snapID, "/subdir/missing.txt", 1<<20)
	require.Error(t, err)

	_, _, err = readResource(repo, snapID, "/subdir", 1<<20)
	require.Error(t, err, "a directory is not a resource")
}

func TestMcpParseRestoreFlags(t *testing.T) {
	_, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	// restores need both switches: the capability and the confinement
	cmd := &Mcp{}
	require.Error(t, cmd.Parse(ctx, []string{"-allow-restore"}))

	cmd = &Mcp{}
	require.Error(t, cmd.Parse(ctx, []string{"-restore-root", t.TempDir()}))

	cmd = &Mcp{}
	require.Error(t, cmd.Parse(ctx, []string{"-allow-restore", "-restore-root", "/no/such/directory"}))

	root := t.TempDir()
	cmd = &Mcp{}
	require.NoError(t, cmd.Parse(ctx, []string{"-allow-restore", "-restore-root", root}))
	require.True(t, cmd.AllowRestore)
	require.Equal(t, root, cmd.RestoreRoot)
}

func TestMcpParseSyncFlag(t *testing.T) {
	_, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	cmd := &Mcp{}
	require.NoError(t, cmd.Parse(ctx, []string{}))
	require.False(t, cmd.AllowSync, "sync must never be on by accident")

	cmd = &Mcp{}
	require.NoError(t, cmd.Parse(ctx, []string{"-allow-sync"}))
	require.True(t, cmd.AllowSync)
}
