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

func TestBackupNoProgressFlag(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, tmpBackupDir, ctx := generateFixtures(t, bufOut, bufErr)

	renderer := stdio.New(ctx)
	renderer.Run()
	defer renderer.Wait()
	defer ctx.Close()

	ctx.MaxConcurrency = 1
	ctx.Stdout = bufOut
	ctx.Stderr = bufErr

	subcommand := &Backup{}
	require.NoError(t, subcommand.Parse(ctx, []string{"-no-progress", tmpBackupDir}))
	require.True(t, subcommand.NoProgress)

	status, err := subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.Contains(t, bufOut.String(), "backup completed")
}

func TestBackupCacheNo(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, tmpBackupDir, ctx := generateFixtures(t, bufOut, bufErr)

	renderer := stdio.New(ctx)
	renderer.Run()
	defer renderer.Wait()
	defer ctx.Close()

	ctx.MaxConcurrency = 1
	ctx.Stdout = bufOut
	ctx.Stderr = bufErr

	subcommand := &Backup{}
	require.NoError(t, subcommand.Parse(ctx, []string{"-cache", "no", tmpBackupDir}))
	require.Equal(t, "no", subcommand.Cache)

	status, err := subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.Contains(t, bufOut.String(), "backup completed")
}

func TestBackupPackfilesMemory(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, tmpBackupDir, ctx := generateFixtures(t, bufOut, bufErr)

	ctx.MaxConcurrency = 1

	subcommand := &Backup{}
	require.NoError(t, subcommand.Parse(ctx, []string{"-packfiles", "memory", tmpBackupDir}))

	status, err := subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
}

func TestBackupExcludeMultiplePatterns(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, tmpBackupDir, ctx := generateFixtures(t, bufOut, bufErr)

	ctx.MaxConcurrency = 1

	subcommand := &Backup{}
	require.NoError(t, subcommand.Parse(ctx, []string{
		"-ignore", "**/foo.txt",
		"-ignore", "**/bar",
		tmpBackupDir,
	}))
	require.Equal(t, []string{"**/foo.txt", "**/bar"}, subcommand.Excludes)

	status, err := subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	out := bufOut.String()
	require.NotContains(t, out, "/foo.txt")
	require.NotContains(t, out, "/another_subdir/bar")
}

func TestBackupIgnoreFile(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, tmpBackupDir, ctx := generateFixtures(t, bufOut, bufErr)

	ignoreFile := filepath.Join(t.TempDir(), "ignore")
	require.NoError(t, os.WriteFile(ignoreFile, []byte("# a comment\n\n**/foo.txt\n"), 0644))

	ctx.MaxConcurrency = 1

	subcommand := &Backup{}
	require.NoError(t, subcommand.Parse(ctx, []string{"-ignore-file", ignoreFile, tmpBackupDir}))
	require.Equal(t, []string{"**/foo.txt"}, subcommand.Excludes)

	status, err := subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.NotContains(t, bufOut.String(), "/foo.txt")
}

func TestBackupIgnoreFileMissing(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	_, tmpBackupDir, ctx := generateFixtures(t, bufOut, bufErr)

	subcommand := &Backup{}
	err := subcommand.Parse(ctx, []string{"-ignore-file", "/no/such/ignore/file", tmpBackupDir})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unable to open excludes file")
}

func TestBackupDryRun(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, tmpBackupDir, ctx := generateFixtures(t, bufOut, bufErr)

	renderer := stdio.New(ctx)
	renderer.Run()
	defer renderer.Wait()
	defer ctx.Close()

	ctx.MaxConcurrency = 1
	ctx.Stdout = bufOut
	ctx.Stderr = bufErr

	subcommand := &Backup{}
	require.NoError(t, subcommand.Parse(ctx, []string{"-dry-run", tmpBackupDir}))
	require.True(t, subcommand.DryRun)

	status, err := subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	// Dry-run still emits per-path lines but must not commit a snapshot.
	require.NotContains(t, bufOut.String(), "backup completed")
}

func TestBackupSecondBackupUsesParent(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, tmpBackupDir, ctx := generateFixtures(t, bufOut, bufErr)

	ctx.MaxConcurrency = 1

	// First backup: nothing to inherit from.
	first := &Backup{}
	require.NoError(t, first.Parse(ctx, []string{tmpBackupDir}))
	status, err := first.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	require.NoError(t, repo.RebuildState())

	// Second backup with default (vfs) cache: it should locate the previous
	// snapshot as a parent and reuse its VFS cache.
	bufOut.Reset()
	second := &Backup{}
	require.NoError(t, second.Parse(ctx, []string{tmpBackupDir}))
	require.Equal(t, "vfs", second.Cache)
	status, err = second.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
}

func TestBackupUnknownScheme(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, _, ctx := generateFixtures(t, bufOut, bufErr)

	ctx.MaxConcurrency = 1

	subcommand := &Backup{}
	require.NoError(t, subcommand.Parse(ctx, []string{"definitely-not-a-scheme://nope"}))

	status, err := subcommand.Execute(ctx, repo)
	require.Error(t, err)
	require.Equal(t, 1, status)
	require.Contains(t, err.Error(), "failed to create an importer")
}

func TestBackupSourceResolution(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, tmpBackupDir, ctx := generateFixtures(t, bufOut, bufErr)
	ctx.Config = config.NewConfig()
	ctx.Config.Sources["mysrc"] = map[string]string{"location": "fs://" + tmpBackupDir}

	ctx.MaxConcurrency = 1

	subcommand := &Backup{}
	require.NoError(t, subcommand.Parse(ctx, []string{"@mysrc"}))

	status, err := subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
}

func TestBackupSourceUnresolved(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, _, ctx := generateFixtures(t, bufOut, bufErr)
	ctx.Config = config.NewConfig()

	subcommand := &Backup{}
	require.NoError(t, subcommand.Parse(ctx, []string{"@ghost"}))

	status, err := subcommand.Execute(ctx, repo)
	require.Error(t, err)
	require.Equal(t, 1, status)
	require.Contains(t, err.Error(), "could not resolve importer")
}

func TestBackupSourceEmptyLocation(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, _, ctx := generateFixtures(t, bufOut, bufErr)
	ctx.Config = config.NewConfig()
	// Source exists but resolves to an empty location: the @ resolution
	// succeeds, but importer creation then fails on the bad location.
	ctx.Config.Sources["nolocsrc"] = map[string]string{"option": "value"}

	subcommand := &Backup{}
	require.NoError(t, subcommand.Parse(ctx, []string{"@nolocsrc"}))

	status, err := subcommand.Execute(ctx, repo)
	require.Error(t, err)
	require.Equal(t, 1, status)
	require.Contains(t, err.Error(), "failed to create an importer")
}
