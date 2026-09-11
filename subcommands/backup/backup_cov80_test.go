package backup

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/PlakarKorp/plakar/config"
	"github.com/PlakarKorp/plakar/ui/stdio"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Parse branches not already exercised elsewhere.
// ---------------------------------------------------------------------------

// TestCov80BackupParseExplicitSourceKept verifies an explicit positional source
// is preserved verbatim (the non-default Sources branch of Parse).
func TestCov80BackupParseExplicitSourceKept(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	_, tmpBackupDir, ctx := generateFixtures(t, bufOut, bufErr)
	t.Cleanup(ctx.Close)

	cmd := &Backup{}
	require.NoError(t, cmd.Parse(ctx, []string{tmpBackupDir}))
	require.Equal(t, []string{tmpBackupDir}, cmd.Sources)
}

// TestCov80BackupParseOptsFlag exercises the -o importer options flag, which
// feeds cmd.Opts and is copied into the per-source options map in DoBackup.
func TestCov80BackupParseOptsFlag(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	_, tmpBackupDir, ctx := generateFixtures(t, bufOut, bufErr)
	t.Cleanup(ctx.Close)

	cmd := &Backup{}
	require.NoError(t, cmd.Parse(ctx, []string{"-o", "key=value", tmpBackupDir}))
	require.Equal(t, "value", cmd.Opts["key"])
}

// TestCov80BackupParseCacheNo pins the -cache flag to the uncached mode.
func TestCov80BackupParseCacheNo(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	_, tmpBackupDir, ctx := generateFixtures(t, bufOut, bufErr)
	t.Cleanup(ctx.Close)

	cmd := &Backup{}
	require.NoError(t, cmd.Parse(ctx, []string{"-cache", "no", tmpBackupDir}))
	require.Equal(t, "no", cmd.Cache)
}

// ---------------------------------------------------------------------------
// DoBackup branches.
// ---------------------------------------------------------------------------

// TestCov80BackupCacheNoEndToEnd runs a real backup with -cache no, which skips
// the vfs parent-lookup branch in DoBackup (the cmd.Cache != "vfs" path).
func TestCov80BackupCacheNoEndToEnd(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, tmpBackupDir, ctx := generateFixtures(t, bufOut, bufErr)

	renderer := stdio.New(ctx)
	renderer.Run()
	t.Cleanup(func() { renderer.Wait() })
	t.Cleanup(ctx.Close)
	ctx.MaxConcurrency = 1

	cmd := &Backup{}
	require.NoError(t, cmd.Parse(ctx, []string{"-cache", "no", tmpBackupDir}))
	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
}

// TestCov80BackupNoProgress runs a backup with -no-progress; this disables the
// separate stats importer and the FilesystemSummary goroutine in DoBackup.
func TestCov80BackupNoProgress(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, tmpBackupDir, ctx := generateFixtures(t, bufOut, bufErr)

	renderer := stdio.New(ctx)
	renderer.Run()
	t.Cleanup(func() { renderer.Wait() })
	t.Cleanup(ctx.Close)
	ctx.MaxConcurrency = 1

	cmd := &Backup{}
	require.NoError(t, cmd.Parse(ctx, []string{"-no-progress", tmpBackupDir}))
	require.True(t, cmd.NoProgress)
	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
}

// TestCov80BackupAtSourceInheritsOptions drives the @source resolution branch
// where the configured source carries extra options that are inherited into the
// importer option map (the "for k, v := range remote" loop in DoBackup).
func TestCov80BackupAtSourceInheritsOptions(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, tmpBackupDir, ctx := generateFixtures(t, bufOut, bufErr)

	renderer := stdio.New(ctx)
	renderer.Run()
	t.Cleanup(func() { renderer.Wait() })
	t.Cleanup(ctx.Close)
	ctx.MaxConcurrency = 1
	ctx.Config = config.NewConfig()

	// A configured source whose location points at the backup dir, plus an
	// extra option that must be inherited by the importer options.
	ctx.Config.Sources["good"] = map[string]string{
		"location": "fs:" + tmpBackupDir,
		"extra":    "inherited",
	}

	cmd := &Backup{}
	require.NoError(t, cmd.Parse(ctx, []string{"@good"}))
	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
}

// TestCov80BackupDryRunNoProgress combines -dry-run with -no-progress to drive
// the dryrun path (progress/ack helpers) without the stats goroutine.
func TestCov80BackupDryRunNoProgress(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, tmpBackupDir, ctx := generateFixtures(t, bufOut, bufErr)

	renderer := stdio.New(ctx)
	renderer.Run()
	t.Cleanup(func() { renderer.Wait() })
	t.Cleanup(ctx.Close)
	ctx.MaxConcurrency = 1

	cmd := &Backup{}
	require.NoError(t, cmd.Parse(ctx, []string{"-dry-run", "-no-progress", tmpBackupDir}))
	status, err, _, _ := cmd.DoBackup(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	// A dry run must not produce a snapshot.
	count := 0
	for _, err := range repo.ListSnapshots() {
		require.NoError(t, err)
		count++
	}
	require.Equal(t, 0, count)
}

// TestCov80BackupDryRunSymlinkAndDir builds a source tree that contains a
// directory, a regular file and a symlink, then dry-runs it. This drives the
// directory / file / symlink Ok branches of dryrun's record switch that the
// default fixture (no symlink) does not reach.
func TestCov80BackupDryRunSymlinkAndDir(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, _, ctx := generateFixtures(t, bufOut, bufErr)

	renderer := stdio.New(ctx)
	renderer.Run()
	t.Cleanup(func() { renderer.Wait() })
	t.Cleanup(ctx.Close)
	ctx.MaxConcurrency = 1

	src := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(src, "dir"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(src, "dir", "file.txt"), []byte("content"), 0o644))
	// A symlink so the record.Target != "" branch in dryrun is hit.
	if err := os.Symlink(filepath.Join(src, "dir", "file.txt"), filepath.Join(src, "link")); err != nil {
		t.Skipf("symlinks unsupported on this platform: %v", err)
	}

	cmd := &Backup{}
	require.NoError(t, cmd.Parse(ctx, []string{"-dry-run", "fs:" + src}))
	status, err, _, _ := cmd.DoBackup(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
}
