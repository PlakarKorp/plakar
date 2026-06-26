package repair

import (
	"bytes"
	"os"
	"testing"

	_ "github.com/PlakarKorp/integration-fs/exporter"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

func init() {
	os.Setenv("TZ", "UTC")
}

func simpleFiles() []ptesting.MockFile {
	return []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockFile("subdir/a.txt", 0644, "alpha"),
		ptesting.NewMockFile("subdir/b.txt", 0644, "bravo"),
	}
}

func TestParseDefault(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	_, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)

	cmd := &Repair{}
	err := cmd.Parse(ctx, []string{})
	require.NoError(t, err)
	require.False(t, cmd.Apply)
}

func TestParseApplyFlag(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	_, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)

	cmd := &Repair{}
	err := cmd.Parse(ctx, []string{"-apply"})
	require.NoError(t, err)
	require.True(t, cmd.Apply)
}

func TestParseCapturesSecret(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	_, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)

	secret := []byte("0123456789abcdef")
	ctx.SetSecret(secret)

	cmd := &Repair{}
	err := cmd.Parse(ctx, []string{})
	require.NoError(t, err)
	require.Equal(t, secret, cmd.RepositorySecret)
}

func TestExecuteDryRunEmptyRepo(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)

	cmd := &Repair{}
	require.NoError(t, cmd.Parse(ctx, []string{}))

	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.Contains(t, bufOut.String(), "no repairs needed")
}

func TestExecuteDryRunPopulatedRepoHealthy(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)
	ptesting.GenerateSnapshot(t, repo, simpleFiles())

	cmd := &Repair{}
	require.NoError(t, cmd.Parse(ctx, []string{}))

	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	// A healthy repository with snapshots has every packfile referenced by a
	// known state, so nothing should be reported as missing.
	require.Contains(t, bufOut.String(), "no repairs needed")
}

func TestExecuteApplyHealthyRepo(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)
	ptesting.GenerateSnapshot(t, repo, simpleFiles())

	cmd := &Repair{}
	require.NoError(t, cmd.Parse(ctx, []string{"-apply"}))
	require.True(t, cmd.Apply)

	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	// On a healthy repo the -apply path finds no missing states, so it neither
	// reports "no repairs needed" nor any "repairing" lines.
	require.NotContains(t, bufOut.String(), "repairing missing state")
}
