package info

import (
	"bytes"
	"encoding/hex"
	"testing"

	_ "github.com/PlakarKorp/integrations/fs/exporter"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

// Parse rejects more than one positional argument.
func TestInfoCov80ParseTooManyArgs(t *testing.T) {
	t.Parallel()
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	_ = repo
	cmd := &Info{}
	err := cmd.Parse(ctx, []string{"a", "b"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "too many arguments")
}

// executeSnapshot against a real snapshot prints the full header dump,
// including the VFS / Importer / Context / Summary sections.
func TestInfoCov80SnapshotFullDump(t *testing.T) {
	t.Parallel()
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	indexID := snap.Header.GetIndexID()
	cmd := &Info{}
	require.NoError(t, cmd.Parse(ctx, []string{hex.EncodeToString(indexID[:])}))
	require.False(t, cmd.Errors)

	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	out := bufOut.String()
	require.Contains(t, out, "SnapshotID:")
	require.Contains(t, out, "VFS:")
	require.Contains(t, out, "Importer:")
	require.Contains(t, out, "Context:")
	require.Contains(t, out, "Summary:")
	require.Contains(t, out, " - Files:")
	require.Contains(t, out, " - Directories:")
}

// executeErrors scoped to a sub-path of a clean snapshot returns 0 with no
// error lines (the fs.Errors iterator yields nothing).
func TestInfoCov80ErrorsSubpathClean(t *testing.T) {
	t.Parallel()
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	indexID := snap.Header.GetIndexID()
	cmd := &Info{}
	require.NoError(t, cmd.Parse(ctx, []string{"-errors",
		hex.EncodeToString(indexID[:]) + ":/subdir"}))
	require.True(t, cmd.Errors)

	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
}

// executeRepository against an unencrypted repo with multiple snapshots
// prints the Snapshots count and logical/storage sizes.
func TestInfoCov80RepositoryMultiSnapshot(t *testing.T) {
	t.Parallel()
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)

	s1 := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockFile("one.txt", 0644, "one"),
	})
	s1.Close()
	s2 := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockFile("two.txt", 0644, "two"),
	})
	s2.Close()

	cmd := &Info{}
	require.NoError(t, cmd.Parse(ctx, []string{}))
	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	out := bufOut.String()
	require.Contains(t, out, "Snapshots: 2")
	require.Contains(t, out, "Chunking:")
	require.Contains(t, out, "Hashing:")
	require.Contains(t, out, "Storage size:")
}
