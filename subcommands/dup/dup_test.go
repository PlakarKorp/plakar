package dup

import (
	"bytes"
	"encoding/hex"
	"os"
	"testing"

	_ "github.com/PlakarKorp/integration-fs/exporter"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

func init() {
	os.Setenv("TZ", "UTC")
}

func generateSnapshot(t *testing.T, bufOut, bufErr *bytes.Buffer) (*repository.Repository, *snapshot.Snapshot, *appcontext.AppContext) {
	repo, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
	})
	return repo, snap, ctx
}

func TestParseNoArgsErrors(t *testing.T) {
	ctx := appcontext.NewAppContext()
	defer ctx.Close()

	cmd := &Dup{}
	err := cmd.Parse(ctx, []string{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "at least one parameter is required")
}

func TestParseStoresSnapshotIDs(t *testing.T) {
	ctx := appcontext.NewAppContext()
	defer ctx.Close()

	cmd := &Dup{}
	err := cmd.Parse(ctx, []string{"aabbcc", "ddeeff:/path"})
	require.NoError(t, err)
	require.Equal(t, []string{"aabbcc", "ddeeff:/path"}, cmd.SnapshotIDS)
}

// Lookup should return a *Dup for the registered "dup" command.
func TestLookupRegistersDup(t *testing.T) {
	subcommand, _, _ := subcommands.Lookup([]string{"dup", "aabbcc"})
	require.NotNil(t, subcommand)
	_, ok := subcommand.(*Dup)
	require.True(t, ok)
}

// Execute against a real snapshot duplicates it without error.
func TestExecuteDuplicatesSnapshot(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	indexID := snap.Header.GetIndexID()
	cmd := &Dup{}
	require.NoError(t, cmd.Parse(ctx, []string{hex.EncodeToString(indexID[:])}))

	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
}

// Execute with a bogus snapshot id exercises the error/continue path and still
// returns status 0 (errors are only counted, not returned).
func TestExecuteBadSnapshotID(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	cmd := &Dup{}
	require.NoError(t, cmd.Parse(ctx, []string{"deadbeefdeadbeef"}))

	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.Contains(t, bufErr.String(), "deadbeefdeadbeef")
}
