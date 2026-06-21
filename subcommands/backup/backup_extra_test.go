package backup

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/PlakarKorp/kloset/locate"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/ui/stdio"
	"github.com/stretchr/testify/require"
)

// runBackup is a small wrapper around the standard Parse + Execute flow that
// most tests in this file want. It returns the exit status, the error, and the
// captured stdout buffer.
//
// The stdio renderer is started here too, mirroring the production wiring:
// without it nothing drains the event bus and Backup.Execute deadlocks.
func runBackup(t *testing.T, args []string) (error, *bytes.Buffer, *repository.Repository, *appcontext.AppContext) {
	t.Helper()
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, tmpBackupDir, ctx := generateFixtures(t, bufOut, bufErr)

	renderer := stdio.New(ctx)
	renderer.Run()
	t.Cleanup(func() { renderer.Wait() })
	t.Cleanup(ctx.Close)

	ctx.MaxConcurrency = 1
	ctx.Stdout = bufOut
	ctx.Stderr = bufErr

	allArgs := append(args, tmpBackupDir)
	return Backup(ctx, repo, allArgs), bufOut, repo, ctx
}

func TestBackupDryRunProducesNoSnapshot(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, tmpBackupDir, ctx := generateFixtures(t, bufOut, bufErr)

	renderer := stdio.New(ctx)
	renderer.Run()
	t.Cleanup(func() { renderer.Wait() })
	t.Cleanup(ctx.Close)
	ctx.MaxConcurrency = 1

	require.NoError(t, Backup(ctx, repo, []string{"-dry-run", tmpBackupDir}))

	// Sanity: the snapshot listing should be empty after a dry run.
	count := 0
	for _, err := range repo.ListSnapshots() {
		require.NoError(t, err)
		count++
	}
	require.Equal(t, 0, count, "dry run should not produce snapshots")
}

func TestBackupNoXattrPropagates(t *testing.T) {
	err, _, _, _ := runBackup(t, []string{"-no-xattr"})
	require.NoError(t, err)
}

func TestBackupNameAndMetadataParseFlags(t *testing.T) {
	args := []string{
		"-name", "snap1",
		"-category", "weekly",
		"-environment", "prod",
		"-perimeter", "datacenter-a",
		"-job", "job-42",
	}
	err, _, repo, _ := runBackup(t, args)
	require.NoError(t, err)

	locateopts := locate.NewDefaultLocateOptions(locate.WithLatest(true))

	require.NoError(t, repo.RebuildState())
	snapshotIDs, err := locate.LocateSnapshotIDs(repo, locateopts)
	require.NoError(t, err, "LocateSnapshotIDs failed")
	require.Len(t, snapshotIDs, 1)

	snap, err := snapshot.Load(repo, snapshotIDs[0])
	require.NoError(t, err)

	require.Equal(t, "snap1", snap.Header.Name)
	require.Equal(t, "weekly", snap.Header.Category)
	require.Equal(t, "prod", snap.Header.Environment)
	require.Equal(t, "datacenter-a", snap.Header.Perimeter)
	require.Equal(t, "job-42", snap.Header.Job)
}

func TestBackupForcedTimestampInPastIsAccepted(t *testing.T) {
	past := time.Now().Add(-24 * time.Hour).UTC().Format(time.RFC3339)
	err, _, _, _ := runBackup(t, []string{"-force-timestamp", past})
	require.NoError(t, err)
}

func TestBackupForcedTimestampInFutureRejected(t *testing.T) {
	future := time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339)
	err, _, _, _ := runBackup(t, []string{"-force-timestamp", future})
	require.Error(t, err)
	require.Contains(t, err.Error(), "future")
}

func TestBackupIgnoreFileFlag(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, tmpBackupDir, ctx := generateFixtures(t, bufOut, bufErr)

	renderer := stdio.New(ctx)
	renderer.Run()
	t.Cleanup(func() { renderer.Wait() })
	t.Cleanup(ctx.Close)
	ctx.MaxConcurrency = 1

	ignoreFile := filepath.Join(t.TempDir(), "ignores")
	// One real entry plus a comment and a blank line to exercise both filters.
	require.NoError(t, os.WriteFile(ignoreFile, []byte("# a comment\n\n**/subdir\n"), 0o600))

	require.NoError(t, Backup(ctx, repo, []string{"-ignore-file", ignoreFile, tmpBackupDir}))
	require.NotContains(t, bufOut.String(), "/subdir/")
}

func TestBackupMultipleIgnoreFileFlags(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, tmpBackupDir, ctx := generateFixtures(t, bufOut, bufErr)

	renderer := stdio.New(ctx)
	renderer.Run()
	t.Cleanup(func() { renderer.Wait() })
	t.Cleanup(ctx.Close)
	ctx.MaxConcurrency = 1

	ignoreDir := t.TempDir()
	macOSIgnoreFile := filepath.Join(ignoreDir, "macos-ignore")
	sourceIgnoreFile := filepath.Join(ignoreDir, "source-ignore")
	require.NoError(t, os.WriteFile(macOSIgnoreFile, []byte(".DS_Store\n"), 0o600))
	require.NoError(t, os.WriteFile(sourceIgnoreFile, []byte("**/subdir\n"), 0o600))

	require.NoError(t, Backup(ctx, repo, []string{
		"-ignore-file", macOSIgnoreFile,
		"-ignore-file", sourceIgnoreFile,
		"-ignore", "**/another_subdir",
		tmpBackupDir,
	}))
	require.NotContains(t, bufOut.String(), "/subdir/")
	require.NotContains(t, bufOut.String(), "/another_subdir/")
}

func TestBackupIgnoreFileMissing(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	_, tmpBackupDir, ctx := generateFixtures(t, bufOut, bufErr)
	t.Cleanup(ctx.Close)
	err := Backup(ctx, nil, []string{"-ignore-file", "/this/does/not/exist", tmpBackupDir})
	require.Error(t, err)
}

func TestLoadIgnoreFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ignores")
	content := "# header\n\npat1\npat2\n  \t\n  \t# leading-space comment is NOT stripped\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	lines, err := LoadIgnoreFile(path)
	require.NoError(t, err)
	require.Equal(t, []string{"pat1", "pat2", "  \t# leading-space comment is NOT stripped"}, lines)
}

func TestLoadIgnoreFileMissing(t *testing.T) {
	_, err := LoadIgnoreFile("/no/such/file")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unable to open")
}

func TestBackupPreHookFailureAbortsBackup(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, tmpBackupDir, ctx := generateFixtures(t, bufOut, bufErr)
	t.Cleanup(ctx.Close)
	ctx.MaxConcurrency = 1

	err := Backup(ctx, repo, []string{"-pre-hook", "exit 7", tmpBackupDir})
	require.Error(t, err)

	require.Error(t, err)
	require.Contains(t, err.Error(), "pre-backup hook failed")
}

func TestBackupPostHookFailureIsNotFatal(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, tmpBackupDir, ctx := generateFixtures(t, bufOut, bufErr)

	renderer := stdio.New(ctx)
	renderer.Run()
	t.Cleanup(func() { renderer.Wait() })
	t.Cleanup(ctx.Close)
	ctx.MaxConcurrency = 1

	require.NoError(t, Backup(ctx, repo, []string{"-post-hook", "exit 9", tmpBackupDir}))
	require.Contains(t, bufOut.String(), "executing hook: exit 9")
}

func TestBackupPackfilesMemory(t *testing.T) {
	err, _, _, _ := runBackup(t, []string{"-packfiles", "memory"})
	require.NoError(t, err)
}

func TestBackupOutputMentionsCompletion(t *testing.T) {
	_, bufOut, _, _ := runBackup(t, nil)
	out := bufOut.String()
	require.True(t, strings.Contains(out, "backup completed"), "missing 'backup completed' line in:\n%s", out)
}
