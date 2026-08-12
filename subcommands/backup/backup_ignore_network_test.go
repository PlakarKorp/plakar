package backup

import (
	"bytes"
	"strings"
	"testing"

	_ "github.com/PlakarKorp/plakar/testing"
	"github.com/PlakarKorp/plakar/ui/stdio"
	"github.com/stretchr/testify/require"
)

func TestBackupIgnoreOnNonLocalImporterWarns(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, _, ctx := generateFixtures(t, bufOut, bufErr)

	renderer := stdio.New(ctx)
	renderer.Run()
	t.Cleanup(func() { renderer.Wait() })
	t.Cleanup(ctx.Close)
	ctx.MaxConcurrency = 1

	cmd := &Backup{}
	require.NoError(t, cmd.Parse(ctx, []string{"-ignore", "**/cache", "mock://source"}))

	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	require.True(t,
		strings.Contains(bufErr.String(), "exclude patterns are applied after traversal"),
		"expected warning on stderr, got:\n%s", bufErr.String())
}

func TestBackupIgnoreOnLocalImporterDoesNotWarn(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, tmpBackupDir, ctx := generateFixtures(t, bufOut, bufErr)

	renderer := stdio.New(ctx)
	renderer.Run()
	t.Cleanup(func() { renderer.Wait() })
	t.Cleanup(ctx.Close)
	ctx.MaxConcurrency = 1

	cmd := &Backup{}
	require.NoError(t, cmd.Parse(ctx, []string{"-ignore", "**/subdir", tmpBackupDir}))

	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	require.False(t,
		strings.Contains(bufErr.String(), "exclude patterns are applied after traversal"),
		"unexpected warning on stderr for local importer, got:\n%s", bufErr.String())
}
