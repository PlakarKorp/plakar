package backup

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/PlakarKorp/kloset/caching"
	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/ui/stdio"
	"github.com/stretchr/testify/require"
)

const (
	failingSnapshotWorkspaceMarkerFile = "partial"
	failingSnapshotWorkspaceErrMessage = "snapshot workspace open failed"
	unexpectedCacheErrMessage          = "unexpected cache constructor call"
	testSnapshotWorkCacheName          = "scan"
	testPackfileTempStorageMode        = "memory"
	testCacheRootFile                  = "cache-root"
)

// runBackup is a small wrapper around the standard Parse + Execute flow that
// most tests in this file want. It returns the exit status, the error, and the
// captured stdout buffer.
//
// The stdio renderer is started here too, mirroring the production wiring:
// without it nothing drains the event bus and Backup.Execute deadlocks.
//
//nolint:staticcheck // ST1008: test helper, error kept in fixed position alongside other outputs
func runBackup(t *testing.T, args []string, mutate func(*Backup)) (int, error, *bytes.Buffer, *appcontext.AppContext) {
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
	cmd := &Backup{}
	if err := cmd.Parse(ctx, allArgs); err != nil {
		return 0, err, bufOut, ctx
	}
	if mutate != nil {
		mutate(cmd)
	}
	status, err := cmd.Execute(ctx, repo)
	return status, err, bufOut, ctx
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

	cmd := &Backup{}
	require.NoError(t, cmd.Parse(ctx, []string{"-dry-run", tmpBackupDir}))
	require.True(t, cmd.DryRun, "DryRun flag should be parsed")

	status, _, _, _ := cmd.DoBackup(ctx, repo)
	require.Equal(t, 0, status)

	// Sanity: the snapshot listing should be empty after a dry run.
	count := 0
	for _, err := range repo.ListSnapshots() {
		require.NoError(t, err)
		count++
	}
	require.Equal(t, 0, count, "dry run should not produce snapshots")
}

func TestBackupNoXattrPropagates(t *testing.T) {
	status, err, _, _ := runBackup(t, []string{"-no-xattr"}, nil)
	require.NoError(t, err)
	require.Equal(t, 0, status)
}

func TestBackupNameAndMetadataParseFlags(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	_, tmpBackupDir, ctx := generateFixtures(t, bufOut, bufErr)
	t.Cleanup(ctx.Close)

	args := []string{
		"-name", "snap1",
		"-category", "weekly",
		"-environment", "prod",
		"-perimeter", "datacenter-a",
		"-job", "job-42",
		tmpBackupDir,
	}
	cmd := &Backup{}
	require.NoError(t, cmd.Parse(ctx, args))
	require.Equal(t, "snap1", cmd.Name)
	require.Equal(t, "weekly", cmd.Category)
	require.Equal(t, "prod", cmd.Environment)
	require.Equal(t, "datacenter-a", cmd.Perimeter)
	require.Equal(t, "job-42", cmd.Job)
}

func TestBackupForcedTimestampInPastIsAccepted(t *testing.T) {
	past := time.Now().Add(-24 * time.Hour).UTC().Format(time.RFC3339)
	status, err, _, _ := runBackup(t, []string{"-force-timestamp", past}, nil)
	require.NoError(t, err)
	require.Equal(t, 0, status)
}

func TestBackupForcedTimestampInFutureRejected(t *testing.T) {
	future := time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339)
	_, err, _, _ := runBackup(t, []string{"-force-timestamp", future}, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "future")
}

func TestCleanupPartialSnapshotWorkspaceRemovesSnapshotDir(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	_, _, ctx := generateFixtures(t, bufOut, bufErr)
	t.Cleanup(ctx.Close)

	identifier := objects.RandomMAC()
	ctx.CacheDir = t.TempDir()
	snapshotWorkDir := filepath.Join(ctx.CacheDir, caching.CACHE_VERSION, snapshotWorkCacheName, fmt.Sprintf("%x", identifier))
	require.NoError(t, os.MkdirAll(snapshotWorkDir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(snapshotWorkDir, failingSnapshotWorkspaceMarkerFile), []byte(failingSnapshotWorkspaceMarkerFile), 0o600))

	cleanupPartialSnapshotWorkspace(ctx, identifier)

	_, statErr := os.Stat(snapshotWorkDir)
	require.True(t, os.IsNotExist(statErr), "partial snapshot workspace %q should be removed", snapshotWorkDir)
}

func TestCleanupPartialSnapshotWorkspaceIgnoresUnsetInputs(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	_, _, ctx := generateFixtures(t, bufOut, bufErr)
	t.Cleanup(ctx.Close)

	cleanupPartialSnapshotWorkspace(ctx, objects.RandomMAC())

	ctx.CacheDir = t.TempDir()
	cleanupPartialSnapshotWorkspace(ctx, objects.NilMac)
}

func TestCleanupPartialSnapshotWorkspaceWarnsWhenRemovalFails(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	_, _, ctx := generateFixtures(t, bufOut, bufErr)
	t.Cleanup(ctx.Close)

	cacheRoot := filepath.Join(t.TempDir(), testCacheRootFile)
	require.NoError(t, os.WriteFile(cacheRoot, []byte(testCacheRootFile), 0o600))
	ctx.CacheDir = cacheRoot

	cleanupPartialSnapshotWorkspace(ctx, objects.RandomMAC())
}

func TestBackupCleansPartialSnapshotWorkspaceOnCreateFailure(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, tmpBackupDir, ctx := generateFixtures(t, bufOut, bufErr)
	t.Cleanup(ctx.Close)

	snapshotWorkspaceErr := errors.New(failingSnapshotWorkspaceErrMessage)
	cacheRoot := t.TempDir()
	var snapshotWorkDir string

	ctx.CacheDir = cacheRoot
	ctx.SetCache(caching.NewManager(func(version, name, repoid string, _ caching.Option) (caching.Cache, error) {
		if name != testSnapshotWorkCacheName {
			return nil, errors.New(unexpectedCacheErrMessage)
		}

		snapshotWorkDir = filepath.Join(cacheRoot, version, name, filepath.FromSlash(repoid))
		require.NoError(t, os.MkdirAll(snapshotWorkDir, 0o700))
		require.NoError(t, os.WriteFile(filepath.Join(snapshotWorkDir, failingSnapshotWorkspaceMarkerFile), []byte(failingSnapshotWorkspaceMarkerFile), 0o600))
		return nil, snapshotWorkspaceErr
	}))

	cmd := &Backup{}
	require.NoError(t, cmd.Parse(ctx, []string{"-packfiles", testPackfileTempStorageMode, "-no-progress", tmpBackupDir}))

	status, err := cmd.Execute(ctx, repo)
	require.Equal(t, 1, status)
	require.ErrorIs(t, err, snapshotWorkspaceErr)

	_, statErr := os.Stat(snapshotWorkDir)
	require.True(t, os.IsNotExist(statErr), "partial snapshot workspace %q should be removed after create failure", snapshotWorkDir)
}

func TestBackupTagViaFlag(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	_, tmpBackupDir, ctx := generateFixtures(t, bufOut, bufErr)

	t.Cleanup(ctx.Close)
	cmd := &Backup{}
	require.NoError(t, cmd.Parse(ctx, []string{"-tag", "alpha,beta", tmpBackupDir}))
	require.Equal(t, []string{"alpha", "beta"}, cmd.Tags)
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

	cmd := &Backup{}
	require.NoError(t, cmd.Parse(ctx, []string{"-ignore-file", ignoreFile, tmpBackupDir}))
	require.Contains(t, cmd.Excludes, "**/subdir")
	// Comments and blanks must not leak into the rule set.
	require.NotContains(t, cmd.Excludes, "# a comment")
	require.NotContains(t, cmd.Excludes, "")

	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.NotContains(t, bufOut.String(), "/subdir/")
}

func TestBackupMultipleIgnoreFileFlags(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, tmpBackupDir, ctx := generateFixtures(t, bufOut, bufErr)

	t.Cleanup(ctx.Close)
	ctx.MaxConcurrency = 1

	ignoreDir := t.TempDir()
	macOSIgnoreFile := filepath.Join(ignoreDir, "macos-ignore")
	sourceIgnoreFile := filepath.Join(ignoreDir, "source-ignore")
	require.NoError(t, os.WriteFile(macOSIgnoreFile, []byte(".DS_Store\n"), 0o600))
	require.NoError(t, os.WriteFile(sourceIgnoreFile, []byte("**/subdir\n"), 0o600))

	cmd := &Backup{}
	require.NoError(t, cmd.Parse(ctx, []string{
		"-ignore-file", macOSIgnoreFile,
		"-ignore-file", sourceIgnoreFile,
		"-ignore", "**/another_subdir",
		tmpBackupDir,
	}))
	require.Equal(t, []string{".DS_Store", "**/subdir", "**/another_subdir"}, cmd.Excludes)

	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.NotContains(t, bufOut.String(), "/subdir/")
	require.NotContains(t, bufOut.String(), "/another_subdir/")
}

func TestBackupIgnoreFileMissing(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	_, tmpBackupDir, ctx := generateFixtures(t, bufOut, bufErr)
	t.Cleanup(ctx.Close)
	cmd := &Backup{}
	err := cmd.Parse(ctx, []string{"-ignore-file", "/this/does/not/exist", tmpBackupDir})
	require.Error(t, err)
}

func TestBackupPreHookFailureAbortsBackup(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, tmpBackupDir, ctx := generateFixtures(t, bufOut, bufErr)
	t.Cleanup(ctx.Close)
	ctx.MaxConcurrency = 1

	cmd := &Backup{}
	require.NoError(t, cmd.Parse(ctx, []string{tmpBackupDir}))
	cmd.PreHook = "exit 7"

	status, err := cmd.Execute(ctx, repo)
	require.Error(t, err)
	require.Equal(t, 1, status)
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

	cmd := &Backup{}
	require.NoError(t, cmd.Parse(ctx, []string{tmpBackupDir}))
	cmd.PostHook = "exit 9"

	status, err := cmd.Execute(ctx, repo)
	// Post-hook failure must not flip the overall result.
	require.NoError(t, err)
	require.Equal(t, 0, status)
	// The hook was at least attempted.
	require.Contains(t, bufOut.String(), "executing hook: exit 9")
}

func TestBackupEmptySourcesUsesCWD(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	_, _, ctx := generateFixtures(t, bufOut, bufErr)
	t.Cleanup(ctx.Close)

	ctx.CWD = "/var/empty"
	cmd := &Backup{}
	require.NoError(t, cmd.Parse(ctx, []string{}))
	require.Equal(t, []string{"fs:/var/empty"}, cmd.Sources)
}

func TestBackupCheckFlagParses(t *testing.T) {
	// We cannot exercise the full -check end-to-end without a real cached
	// daemon: the fake cached server returns OK without actually rebuilding
	// state, and the subsequent integrity check fails with "blob not found".
	// So just pin down that the flag is parsed.
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	_, tmpBackupDir, ctx := generateFixtures(t, bufOut, bufErr)
	t.Cleanup(ctx.Close)
	cmd := &Backup{}
	require.NoError(t, cmd.Parse(ctx, []string{"-check", tmpBackupDir}))
	require.True(t, cmd.OptCheck)
}

func TestBackupPackfilesMemory(t *testing.T) {
	status, err, _, _ := runBackup(t, []string{"-packfiles", "memory"}, nil)
	require.NoError(t, err)
	require.Equal(t, 0, status)
}

func TestBackupPropagatesContextCause(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, tmpBackupDir, ctx := generateFixtures(t, bufOut, bufErr)

	t.Cleanup(ctx.Close)
	ctx.MaxConcurrency = 1

	cause := errors.New("packfile temp creation failed")
	ctx.Cancel(cause)

	cmd := &Backup{}
	require.NoError(t, cmd.Parse(ctx, []string{tmpBackupDir}))

	status, err := cmd.Execute(ctx, repo)
	require.Error(t, err)
	require.Equal(t, 1, status)
	require.NotErrorIs(t, err, context.Canceled)
	require.ErrorIs(t, err, cause)
}

func TestBackupParsesMultipleIgnoreFlags(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	_, tmpBackupDir, ctx := generateFixtures(t, bufOut, bufErr)
	t.Cleanup(ctx.Close)
	cmd := &Backup{}
	require.NoError(t, cmd.Parse(ctx, []string{"-ignore", "*.tmp", "-ignore", "*.log", tmpBackupDir}))
	require.Contains(t, cmd.Excludes, "*.tmp")
	require.Contains(t, cmd.Excludes, "*.log")
}

func TestBackupOutputMentionsCompletion(t *testing.T) {
	_, _, bufOut, _ := runBackup(t, nil, nil)
	out := bufOut.String()
	require.True(t, strings.Contains(out, "backup completed"), "missing 'backup completed' line in:\n%s", out)
}
