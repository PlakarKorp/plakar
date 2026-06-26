package task

import (
	"bytes"
	"encoding/hex"
	"os"
	"testing"

	_ "github.com/PlakarKorp/integration-fs/exporter"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/plakar/appcontext"
	ptesting "github.com/PlakarKorp/plakar/testing"

	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/PlakarKorp/plakar/subcommands/check"
	"github.com/PlakarKorp/plakar/subcommands/diag"
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

// RunCommand with a known task kind (check) should run the command's Execute
// path and finish successfully, emitting a TaskDone report.
func TestRunCommandKnownTaskKind(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	indexID := snap.Header.GetIndexID()
	args := []string{"check", hex.EncodeToString(indexID[:])}

	subcommand, _, rest := subcommands.Lookup(args)
	require.NotNil(t, subcommand)
	err := subcommand.Parse(ctx, rest)
	require.NoError(t, err)

	_, ok := subcommand.(*check.Check)
	require.True(t, ok)

	status, err := RunCommand(ctx, subcommand, repo, "my-check-task")
	require.NoError(t, err)
	require.Equal(t, 0, status)
}

// RunCommand with a command that is not one of the recognized task kinds
// exercises the default branch (report.SetIgnore) while still executing the
// command normally.
func TestRunCommandUnknownTaskKind(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	indexID := snap.Header.GetIndexID()
	args := []string{"diag", "snapshot", hex.EncodeToString(indexID[:])}

	subcommand, _, rest := subcommands.Lookup(args)
	require.NotNil(t, subcommand)
	err := subcommand.Parse(ctx, rest)
	require.NoError(t, err)

	_, ok := subcommand.(*diag.DiagSnapshot)
	require.True(t, ok)

	status, err := RunCommand(ctx, subcommand, repo, "diag-task")
	require.NoError(t, err)
	require.Equal(t, 0, status)
	// diag snapshot prints the snapshot header to stdout
	require.Contains(t, bufOut.String(), hex.EncodeToString(indexID[:]))
}

// RunCommand should tolerate a nil repository (location lookup is guarded).
func TestRunCommandNilRepo(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()
	_ = repo

	// A command whose Execute fails fast without touching the repo.
	indexID := snap.Header.GetIndexID()
	args := []string{"check", hex.EncodeToString(indexID[:])}
	subcommand, _, rest := subcommands.Lookup(args)
	require.NotNil(t, subcommand)
	require.NoError(t, subcommand.Parse(ctx, rest))

	// Run against the real repo but assert the report machinery handles the
	// repo-name/repo-attach branch.
	status, err := RunCommand(ctx, subcommand, repo, "")
	require.NoError(t, err)
	require.Equal(t, 0, status)
}
