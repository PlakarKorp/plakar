package backup

import (
	"bytes"
	"testing"

	"github.com/PlakarKorp/plakar/config"
	"github.com/PlakarKorp/plakar/ui/stdio"
	"github.com/stretchr/testify/require"
)

// ---------- tagFlags.Set: specified twice is an error ----------

func TestCov2TagsSpecifiedTwiceErrors(t *testing.T) {
	// Drive tagFlags directly: a second Set is an error.
	var tf tagFlags
	require.NoError(t, tf.Set("a,b"))
	require.Error(t, tf.Set("c"))
	require.Equal(t, []string{"a", "b"}, tf.asList())

	var empty tagFlags
	require.Equal(t, []string{}, empty.asList())
	require.Equal(t, "a,b", tf.String())
}

// ---------- @source resolution error paths ----------

func TestCov2BackupUnknownAtSource(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, _, ctx := generateFixtures(t, bufOut, bufErr)

	renderer := stdio.New(ctx)
	renderer.Run()
	t.Cleanup(func() { renderer.Wait() })
	t.Cleanup(ctx.Close)
	ctx.MaxConcurrency = 1
	ctx.Config = config.NewConfig()

	cmd := &Backup{}
	require.NoError(t, cmd.Parse(ctx, []string{"@nope"}))
	status, err := cmd.Execute(ctx, repo)
	require.Equal(t, 1, status)
	require.Error(t, err)
	require.Contains(t, err.Error(), "could not resolve importer")
}

func TestCov2BackupAtSourceResolvedSucceeds(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, tmpBackupDir, ctx := generateFixtures(t, bufOut, bufErr)

	renderer := stdio.New(ctx)
	renderer.Run()
	t.Cleanup(func() { renderer.Wait() })
	t.Cleanup(ctx.Close)
	ctx.MaxConcurrency = 1
	ctx.Config = config.NewConfig()

	// a configured source whose location points at the backup dir
	ctx.Config.Sources["good"] = map[string]string{"location": "fs:" + tmpBackupDir}

	cmd := &Backup{}
	require.NoError(t, cmd.Parse(ctx, []string{"@good"}))
	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
}

// ---------- importer creation failure (unknown scheme) ----------

func TestCov2BackupBadImporterScheme(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, _, ctx := generateFixtures(t, bufOut, bufErr)

	renderer := stdio.New(ctx)
	renderer.Run()
	t.Cleanup(func() { renderer.Wait() })
	t.Cleanup(ctx.Close)
	ctx.MaxConcurrency = 1

	cmd := &Backup{}
	require.NoError(t, cmd.Parse(ctx, []string{"doesnotexist:///tmp/x"}))
	status, err := cmd.Execute(ctx, repo)
	require.Equal(t, 1, status)
	require.Error(t, err)
}

// ---------- packfiles temp dir on disk ----------

func TestCov2BackupPackfilesOnDisk(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, tmpBackupDir, ctx := generateFixtures(t, bufOut, bufErr)

	renderer := stdio.New(ctx)
	renderer.Run()
	t.Cleanup(func() { renderer.Wait() })
	t.Cleanup(ctx.Close)
	ctx.MaxConcurrency = 1

	pkDir := t.TempDir()
	cmd := &Backup{}
	require.NoError(t, cmd.Parse(ctx, []string{"-packfiles", pkDir, tmpBackupDir}))
	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
}

// ---------- dry-run with excludes exercises dryrun's exclude branch ----------

func TestCov2BackupDryRunWithExcludes(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, tmpBackupDir, ctx := generateFixtures(t, bufOut, bufErr)

	renderer := stdio.New(ctx)
	renderer.Run()
	t.Cleanup(func() { renderer.Wait() })
	t.Cleanup(ctx.Close)
	ctx.MaxConcurrency = 1

	cmd := &Backup{}
	require.NoError(t, cmd.Parse(ctx, []string{"-dry-run", "-ignore", "*.txt", tmpBackupDir}))
	status, err, _, _ := cmd.DoBackup(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
}
