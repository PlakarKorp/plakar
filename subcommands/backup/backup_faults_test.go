package backup

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/PlakarKorp/plakar/config"
	"github.com/PlakarKorp/plakar/ui/stdio"
	"github.com/stretchr/testify/require"
)

// TestFaultBackupEmptyAtSourceUnresolved drives an @-reference to a source that
// is not present in the configuration at all, exercising the
// `remote, ok := ctx.Config.GetSource(...); if !ok` error branch with a config
// that has other (unrelated) sources defined.
func TestFaultBackupEmptyAtSourceUnresolved(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, tmpBackupDir, ctx := generateFixtures(t, bufOut, bufErr)

	renderer := stdio.New(ctx)
	renderer.Run()
	t.Cleanup(func() { renderer.Wait() })
	t.Cleanup(ctx.Close)
	ctx.MaxConcurrency = 1
	ctx.Config = config.NewConfig()
	// An unrelated configured source so the lookup map is non-empty.
	ctx.Config.Sources["other"] = map[string]string{"location": "fs:" + tmpBackupDir}

	cmd := &Backup{}
	require.NoError(t, cmd.Parse(ctx, []string{"@missing"}))
	status, err := cmd.Execute(ctx, repo)
	require.Equal(t, 1, status)
	require.Error(t, err)
	require.Contains(t, err.Error(), "could not resolve importer")
}

// TestFaultBackupPackfilesBadDir points -packfiles at a path under a file (not a
// directory) so os.MkdirTemp fails, exercising DoBackup's
// `tmpDir, err := os.MkdirTemp(...); if err != nil` branch.
func TestFaultBackupPackfilesBadDir(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, tmpBackupDir, ctx := generateFixtures(t, bufOut, bufErr)

	renderer := stdio.New(ctx)
	renderer.Run()
	t.Cleanup(func() { renderer.Wait() })
	t.Cleanup(ctx.Close)
	ctx.MaxConcurrency = 1

	// A path that does not exist and cannot be created as a temp-dir parent.
	bad := filepath.Join(tmpBackupDir, "subdir", "dummy.txt", "not-a-dir")

	cmd := &Backup{}
	require.NoError(t, cmd.Parse(ctx, []string{"-packfiles", bad, tmpBackupDir}))
	status, err := cmd.Execute(ctx, repo)
	require.Equal(t, 1, status)
	require.Error(t, err)
}

