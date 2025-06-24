package create

import (
	"fmt"
	"os"
	"syscall"
	"testing"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	_ "github.com/PlakarKorp/plakar/connectors/fs/storage"
	"github.com/creack/pty"
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

	args := []string{"--no-encryption", "--hashing", "SHA256"}

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
	args := []string{"--no-encryption", "--no-compression"}

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
	args := []string{"--no-encryption"}

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

func _TestExecuteCmdCreateDefaultWeakPassword(t *testing.T) {
	tmpRepoDirRoot, err := os.MkdirTemp("", "tmp_repo")
	require.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(tmpRepoDirRoot)
	})
	// Create a pipe to capture stdout
	pty, tty, err := pty.Open()
	require.NoError(t, err)
	defer pty.Close()
	defer tty.Close()

	// Duplicate the tty file descriptor to syscall.Stdin (fd 0)
	originalStdin, err := syscall.Dup(syscall.Stdin)
	require.NoError(t, err)
	defer syscall.Dup2(originalStdin, syscall.Stdin)

	err = syscall.Dup2(int(tty.Fd()), syscall.Stdin)
	require.NoError(t, err)

	ctx := appcontext.NewAppContext()
	defer ctx.Close()

	repo, err := repository.Inexistent(ctx.GetInner(), map[string]string{"location": tmpRepoDirRoot + "/repo"})
	require.NoError(t, err)
	args := []string{}

	subcommand := &Create{}
	err = subcommand.Parse(ctx, args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	input := "azerty\nazerty\nazerty\n"
	_, err = pty.WriteString(input)
	require.NoError(t, err)

	status, err := subcommand.Execute(ctx, repo)
	require.Error(t, err, "password is too weak")
	require.Equal(t, 1, status)

	// try again with authorization to use weak passphrase
	args = []string{"--weak-passphrase"}

	subcommand = &Create{}
	err = subcommand.Parse(ctx, args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	input = "azerty\nazerty\n"
	_, err = pty.WriteString(input)
	require.NoError(t, err)

	status, err = subcommand.Execute(ctx, repo)
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

func TestExecuteCmdCreateDefaultWithEnvPassphrase(t *testing.T) {
	tmpRepoDirRoot, err := os.MkdirTemp("", "tmp_repo")
	require.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(tmpRepoDirRoot)
	})
	ctx := appcontext.NewAppContext()
	defer ctx.Close()

	t.Setenv("PLAKAR_PASSPHRASE", "")

	args := []string{}

	subcommand := &Create{}
	err = subcommand.Parse(ctx, args)
	require.Error(t, err, "can't encrypt the repository with an empty passphrase")

	t.Setenv("PLAKAR_PASSPHRASE", "aZeRtY123456$#@!@")
	err = subcommand.Parse(ctx, args)
	require.NoError(t, err)

	_, err = os.Stat(fmt.Sprintf("%s/repo/CONFIG", tmpRepoDirRoot))
	require.NotNil(t, err)
}
