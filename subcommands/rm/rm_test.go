package rm

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"testing"

	_ "github.com/PlakarKorp/integrations/fs/exporter"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/plakar/appcontext"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

func init() {
	os.Setenv("TZ", "UTC")
}

func generateSnapshot(t *testing.T, bufOut *bytes.Buffer, bufErr *bytes.Buffer) (*repository.Repository, *snapshot.Snapshot, *appcontext.AppContext) {
	repo, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockDir("another_subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
		ptesting.NewMockFile("subdir/foo.txt", 0644, "hello foo"),
		ptesting.NewMockFile("subdir/to_exclude", 0644, "*/subdir/to_exclude\n"),
		ptesting.NewMockFile("another_subdir/bar.txt", 0644, "hello bar"),
	})
	return repo, snap, ctx
}

func TestExecuteCmdRmDefault(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	// Need -apply to actually delete (otherwise it's a dry-run plan).
	args := []string{"-latest", "-apply"}

	subcommand := &Rm{}
	err := subcommand.Parse(ctx, args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	status, err := subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	output := bufOut.String()
	require.Contains(t, output, fmt.Sprintf("info: rm: removal of %s completed successfully", hex.EncodeToString(snap.Header.GetIndexShortID())))
}
func TestExecuteCmdRmWithSnapshot(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	args := []string{"-apply", hex.EncodeToString(snap.Header.GetIndexShortID())}

	subcommand := &Rm{}
	err := subcommand.Parse(ctx, args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	status, err := subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	output := bufOut.String()
	require.Contains(t, output,
		fmt.Sprintf("info: rm: removal of %s completed successfully", hex.EncodeToString(snap.Header.GetIndexShortID())),
	)
	// sanity: no dry-run text
	require.NotContains(t, output, "rm: would remove these")
}

func TestRmParseNoFilterRejected(t *testing.T) {
	_, snap, ctx := generateSnapshot(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil))
	defer snap.Close()

	cmd := &Rm{}
	err := cmd.Parse(ctx, []string{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "not going to remove everything")
}

func TestRmExecuteNoMatches(t *testing.T) {
	// A filter that matches nothing logs an informational line and returns 0.
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	cmd := &Rm{}
	require.NoError(t, cmd.Parse(ctx, []string{"deadbeefdeadbeef"}))
	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.Contains(t, bufOut.String()+bufErr.String(), "no snapshots matched")
}

func TestRm_DryRun_MultipleSnapshotsSorted(t *testing.T) {
	// With two matching snapshots the dry-run plan exercises the sort comparator
	// across distinct timestamps.
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)
	snap1 := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockFile("a.txt", 0644, "a"),
	})
	defer snap1.Close()
	snap2 := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockFile("b.txt", 0644, "b"),
	})
	defer snap2.Close()

	id1 := snap1.Header.GetIndexID()
	id2 := snap2.Header.GetIndexID()
	cmd := &Rm{}
	require.NoError(t, cmd.Parse(ctx, []string{
		hex.EncodeToString(id1[:]),
		hex.EncodeToString(id2[:]),
	}))

	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.Contains(t, bufOut.String(), "rm: would remove these 2 snapshot(s)")
}

func TestRm_DryRun_ShowsPlan(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	args := []string{"-latest"} // no -apply → plan only

	subcommand := &Rm{}
	require.NoError(t, subcommand.Parse(ctx, args))
	_, err := subcommand.Execute(ctx, repo)
	require.NoError(t, err)

	out := bufOut.String()
	require.Contains(t, out, "rm: would remove these 1 snapshot(s), run with -apply to proceed")
	require.NotContains(t, out, "rm: removal of") // no actual deletion
}
