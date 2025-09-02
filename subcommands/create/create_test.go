package create

import (
	"fmt"
	"os"
	"testing"

	_ "github.com/PlakarKorp/integration-fs/storage"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/stretchr/testify/require"
)

func init() {
	os.Setenv("TZ", "UTC")
}

func TestExecuteCmdCreateDefaultWithHashing(t *testing.T) {
	tmpRepoDirRoot, err := os.MkdirTemp("", "tmp_repo")
	require.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(tmpRepoDirRoot)
	})
	ctx := appcontext.NewAppContext()
	defer ctx.Close()

	repo, err := repository.Inexistent(ctx.GetInner(), map[string]string{"location": tmpRepoDirRoot + "/repo"})
	require.NoError(t, err)

	args := []string{"-plaintext"}

	subcommand := &Create{}
	err = subcommand.Parse(ctx, args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	status, err := subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	_, err = os.Stat(fmt.Sprintf("%s/repo/CONFIG", tmpRepoDirRoot))
	require.NoError(t, err)
}

func TestExecuteCmdCreateDefaultWithoutCompression(t *testing.T) {
	tmpRepoDirRoot, err := os.MkdirTemp("", "tmp_repo")
	require.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(tmpRepoDirRoot)
	})
	ctx := appcontext.NewAppContext()
	defer ctx.Close()

	repo, err := repository.Inexistent(ctx.GetInner(), map[string]string{"location": tmpRepoDirRoot + "/repo"})
	require.NoError(t, err)
	args := []string{"-plaintext"}

	subcommand := &Create{}
	err = subcommand.Parse(ctx, args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	status, err := subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	_, err = os.Stat(fmt.Sprintf("%s/repo/CONFIG", tmpRepoDirRoot))
	require.NoError(t, err)
}

func TestExecuteCmdCreateDefaultWithoutEncryption(t *testing.T) {
	tmpRepoDirRoot, err := os.MkdirTemp("", "tmp_repo")
	require.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(tmpRepoDirRoot)
	})
	ctx := appcontext.NewAppContext()
	defer ctx.Close()

	repo, err := repository.Inexistent(ctx.GetInner(), map[string]string{"location": tmpRepoDirRoot + "/repo"})
	require.NoError(t, err)
	args := []string{"-plaintext"}

	subcommand := &Create{}
	err = subcommand.Parse(ctx, args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	status, err := subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	_, err = os.Stat(fmt.Sprintf("%s/repo/CONFIG", tmpRepoDirRoot))
	require.NoError(t, err)
}

func TestExecuteCmdCreateDefaultWithKeyfile(t *testing.T) {
	tmpRepoDirRoot, err := os.MkdirTemp("", "tmp_repo")
	require.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(tmpRepoDirRoot)
	})
	ctx := appcontext.NewAppContext()
	defer ctx.Close()

	repo, err := repository.Inexistent(ctx.GetInner(), map[string]string{"location": tmpRepoDirRoot + "/repo"})
	require.NoError(t, err)
	ctx.KeyFromFile = "aZeRtY123456$#@!@"
	args := []string{}

	subcommand := &Create{}
	err = subcommand.Parse(ctx, args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	status, err := subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	_, err = os.Stat(fmt.Sprintf("%s/repo/CONFIG", tmpRepoDirRoot))
	require.NoError(t, err)
}
