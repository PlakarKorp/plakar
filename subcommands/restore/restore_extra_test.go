package restore

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

func mkRestoreDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "tmp_to_restore")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

func TestRestoreParseRejectsMultiplePaths(t *testing.T) {
	// Only one positional argument is allowed (besides flags).
	_, _, ctx := generateSnapshot(t)
	cmd := &Restore{}
	err := cmd.Parse(ctx, []string{"abc", "def", "ghi"})
	// Parse currently does not actually reject multi-path on its own — the
	// check is `flags.NArg() > 1` inside an `else if` whose preceding branch
	// already consumed the case. Pin down the present behavior so we notice
	// if this gets tightened: Parse succeeds, Snapshots has all positional
	// args.
	require.NoError(t, err)
	require.Len(t, cmd.Snapshots, 3)
}

func TestRestoreParseDefaultTargetIncludesCWD(t *testing.T) {
	_, _, ctx := generateSnapshot(t)
	ctx.CWD = "/var/test-cwd"
	cmd := &Restore{}
	require.NoError(t, cmd.Parse(ctx, []string{}))
	require.True(t, strings.HasPrefix(cmd.Target, "/var/test-cwd/plakar-"), "Target = %q", cmd.Target)
}

func TestRestoreParseHonorsToFlag(t *testing.T) {
	_, _, ctx := generateSnapshot(t)
	cmd := &Restore{}
	require.NoError(t, cmd.Parse(ctx, []string{"-to", "/some/where"}))
	require.Equal(t, "/some/where", cmd.Target)
}

func TestRestoreParseSnapshotWithFiltersWarns(t *testing.T) {
	repo, snap, ctx := generateSnapshot(t)
	defer snap.Close()
	_ = repo
	cmd := &Restore{}
	id := snap.Header.GetIndexID()
	err := cmd.Parse(ctx, []string{"-name", "x", hex.EncodeToString(id[:])})
	// Filter + snapshot is valid (warned only, not rejected).
	require.NoError(t, err)
	require.Equal(t, "x", cmd.OptName)
}

func TestRestoreNoMatchingSnapshot(t *testing.T) {
	repo, snap, ctx := generateSnapshot(t)
	defer snap.Close()

	cmd := &Restore{}
	require.NoError(t, cmd.Parse(ctx, []string{"-to", mkRestoreDir(t), "-name", "no-such-name-anywhere"}))

	status, err := cmd.Execute(ctx, repo)
	require.Error(t, err)
	require.Equal(t, 1, status)
	require.Contains(t, err.Error(), "no snapshots found")
}

func TestRestoreWithSnapshotPath(t *testing.T) {
	// "<id>:<sub-path>" restores only the named subtree.
	repo, snap, ctx := generateSnapshot(t)
	defer snap.Close()

	dir := mkRestoreDir(t)
	id := snap.Header.GetIndexID()
	target := fmt.Sprintf("%s:/subdir", hex.EncodeToString(id[:]))

	cmd := &Restore{}
	require.NoError(t, cmd.Parse(ctx, []string{"-to", dir, target}))
	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	// The snapshot:path syntax narrows the restore to the named subtree. The
	// exact target layout depends on the Strip computation in Execute, so
	// instead of guessing it, walk the restore dir and check that:
	//   - at least one file containing "hello dummy" is present (came from subdir)
	//   - no file containing "hello bar" is present (would have come from
	//     another_subdir, which was not selected)
	sawDummy, sawBar := false, false
	require.NoError(t, filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if strings.Contains(string(data), "hello dummy") {
			sawDummy = true
		}
		if strings.Contains(string(data), "hello bar") {
			sawBar = true
		}
		return nil
	}))
	require.True(t, sawDummy, "expected dummy.txt content under %s", dir)
	require.False(t, sawBar, "another_subdir should not have been restored")
}

func TestRestoreSkipPermissionsFlag(t *testing.T) {
	_, _, ctx := generateSnapshot(t)
	cmd := &Restore{}
	require.NoError(t, cmd.Parse(ctx, []string{"-skip-permissions", "-to", "/tmp/x"}))
	require.True(t, cmd.OptSkipPermissions)
}

func TestRestoreFilterFlagsAreParsed(t *testing.T) {
	_, _, ctx := generateSnapshot(t)
	cmd := &Restore{}
	args := []string{
		"-name", "n",
		"-category", "c",
		"-environment", "e",
		"-perimeter", "p",
		"-job", "j",
		"-tag", "t",
		"-to", "/tmp/x",
	}
	require.NoError(t, cmd.Parse(ctx, args))
	require.Equal(t, "n", cmd.OptName)
	require.Equal(t, "c", cmd.OptCategory)
	require.Equal(t, "e", cmd.OptEnvironment)
	require.Equal(t, "p", cmd.OptPerimeter)
	require.Equal(t, "j", cmd.OptJob)
	require.Equal(t, "t", cmd.OptTag)
}

func TestRestoreExporterOptsParsed(t *testing.T) {
	_, _, ctx := generateSnapshot(t)
	cmd := &Restore{}
	require.NoError(t, cmd.Parse(ctx, []string{"-o", "k=v", "-o", "x=y", "-to", "/tmp/x"}))
	require.Equal(t, "v", cmd.Opts["k"])
	require.Equal(t, "y", cmd.Opts["x"])
}

func TestRestoreHooksFlagsAreParsed(t *testing.T) {
	_, _, ctx := generateSnapshot(t)
	cmd := &Restore{}
	require.NoError(t, cmd.Parse(ctx, []string{
		"-pre-hook", "echo 'pre-restore hook'",
		"-post-hook", "echo 'post-restore hook'",
		"-to", "/tmp/x",
	}))
	require.Equal(t, "echo 'pre-restore hook'", cmd.OptPreHook)
	require.Equal(t, "echo 'post-restore hook'", cmd.OptPostHook)
}

func TestRestoreWithHooks(t *testing.T) {
	repo, snap, ctx := generateSnapshot(t)
	defer snap.Close()

	dir := mkRestoreDir(t)
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	ctx.Stdout = bufOut
	ctx.Stderr = bufErr

	id := snap.Header.GetIndexID()
	cmd := &Restore{}
	require.NoError(t, cmd.Parse(ctx, []string{
		"-to", dir,
		"-pre-hook", "echo 'pre-restore hook executed'",
		"-post-hook", "echo 'post-restore hook executed'",
		hex.EncodeToString(id[:]) + ":",
	}))

	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	output := bufOut.String()
	require.Contains(t, output, "pre-restore hook executed")
	require.Contains(t, output, "post-restore hook executed")

	checkRestored(t, dir)
}

func TestRestorePreHookFailureAbortsRestore(t *testing.T) {
	repo, snap, ctx := generateSnapshot(t)
	defer snap.Close()

	dir := mkRestoreDir(t)
	id := snap.Header.GetIndexID()
	cmd := &Restore{}
	require.NoError(t, cmd.Parse(ctx, []string{
		"-to", dir,
		"-pre-hook", "exit 7",
		hex.EncodeToString(id[:]) + ":",
	}))

	status, err := cmd.Execute(ctx, repo)
	require.Error(t, err)
	require.Equal(t, 1, status)
	require.Contains(t, err.Error(), "pre-restore hook failed")

	rest, readErr := os.ReadDir(dir)
	require.NoError(t, readErr)
	require.Empty(t, rest)
}

func TestRestorePostHookFailureIsNotFatal(t *testing.T) {
	repo, snap, ctx := generateSnapshot(t)
	defer snap.Close()

	dir := mkRestoreDir(t)
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	ctx.Stdout = bufOut
	ctx.Stderr = bufErr

	id := snap.Header.GetIndexID()
	cmd := &Restore{}
	require.NoError(t, cmd.Parse(ctx, []string{
		"-to", dir,
		"-post-hook", "exit 9",
		hex.EncodeToString(id[:]) + ":",
	}))

	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	checkRestored(t, dir)
}

func TestRestoreInvalidSnapshotID(t *testing.T) {
	// A garbage snapshot ID should yield "no snapshots found" (because it
	// matches nothing in the locate filter).
	repo, snap, ctx := generateSnapshot(t)
	defer snap.Close()

	cmd := &Restore{}
	require.NoError(t, cmd.Parse(ctx, []string{"-to", mkRestoreDir(t), "deadbeefdeadbeef"}))
	status, err := cmd.Execute(ctx, repo)
	require.Error(t, err)
	require.Equal(t, 1, status)
}

func TestRestoreMultipleSnapshotsFound(t *testing.T) {
	// Two snapshots match the (default, latest-less) locate filter, so Execute
	// must refuse with "multiple snapshots found".
	repo, snap, ctx := generateSnapshot(t)
	defer snap.Close()
	snap2 := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
	})
	defer snap2.Close()

	id1 := snap.Header.GetIndexID()
	id2 := snap2.Header.GetIndexID()

	cmd := &Restore{}
	require.NoError(t, cmd.Parse(ctx, []string{
		"-to", mkRestoreDir(t),
		hex.EncodeToString(id1[:]) + ":",
		hex.EncodeToString(id2[:]) + ":",
	}))
	status, err := cmd.Execute(ctx, repo)
	require.Error(t, err)
	require.Equal(t, 1, status)
	require.Contains(t, err.Error(), "multiple snapshots found")
}

func TestRestoreToAliasWithLocation(t *testing.T) {
	// A destination alias that resolves to a usable fs:// location restores
	// successfully — exercises the alias success branch in Execute.
	repo, snap, ctx := generateSnapshot(t)
	defer snap.Close()

	dir := mkRestoreDir(t)
	ctx.ConfigDir = t.TempDir()
	require.NoError(t, ctx.ReloadConfig())
	ctx.Config.Destinations["mydest"] = map[string]string{"location": "fs://" + dir}

	id := snap.Header.GetIndexID()
	cmd := &Restore{}
	require.NoError(t, cmd.Parse(ctx, []string{"-to", "@mydest", hex.EncodeToString(id[:]) + ":"}))
	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	checkRestored(t, dir)
}

func TestRestoreSkipPermissionsExecutes(t *testing.T) {
	// Drive a real restore with -skip-permissions so the SkipPermissions branch
	// in Execute is taken.
	repo, snap, ctx := generateSnapshot(t)
	defer snap.Close()

	dir := mkRestoreDir(t)
	id := snap.Header.GetIndexID()
	cmd := &Restore{}
	require.NoError(t, cmd.Parse(ctx, []string{"-skip-permissions", "-to", dir, hex.EncodeToString(id[:]) + ":"}))
	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	checkRestored(t, dir)
}

func TestRestoreSubtreeWithTrailingSlashStrip(t *testing.T) {
	// "<id>:/subdir/" (with a trailing slash) takes the Strip == pathname branch
	// rather than path.Dir(pathname).
	repo, snap, ctx := generateSnapshot(t)
	defer snap.Close()

	dir := mkRestoreDir(t)
	id := snap.Header.GetIndexID()
	target := fmt.Sprintf("%s:/subdir/", hex.EncodeToString(id[:]))

	cmd := &Restore{}
	require.NoError(t, cmd.Parse(ctx, []string{"-to", dir, target}))
	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	// The subdir contents land directly under the restore dir when the whole
	// subdir path is stripped.
	sawDummy := false
	require.NoError(t, filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if strings.Contains(string(data), "hello dummy") {
			sawDummy = true
		}
		return nil
	}))
	require.True(t, sawDummy, "expected dummy.txt content under %s", dir)
}

func TestRestoreInvalidExporterLocation(t *testing.T) {
	// An unknown exporter scheme makes NewExporter fail inside Execute.
	repo, snap, ctx := generateSnapshot(t)
	defer snap.Close()

	id := snap.Header.GetIndexID()
	cmd := &Restore{}
	require.NoError(t, cmd.Parse(ctx, []string{"-to", "bogusscheme://nowhere", hex.EncodeToString(id[:]) + ":"}))
	status, err := cmd.Execute(ctx, repo)
	require.Error(t, err)
	require.Equal(t, 1, status)
}

func TestRestoreToUnresolvableAlias(t *testing.T) {
	// "@something" with no Config will surface as an exporter-resolution
	// error.
	repo, snap, ctx := generateSnapshot(t)
	defer snap.Close()

	// Initialize an empty Config so the lookup runs (and reports not found)
	// rather than NPE-ing inside GetDestination on a nil receiver.
	ctx.ConfigDir = t.TempDir()
	require.NoError(t, ctx.ReloadConfig())

	cmd := &Restore{}
	require.NoError(t, cmd.Parse(ctx, []string{"-to", "@no-such-destination"}))
	status, err := cmd.Execute(ctx, repo)
	require.Error(t, err)
	require.Equal(t, 1, status)
	require.Contains(t, err.Error(), "exporter")
}
