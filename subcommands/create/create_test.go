package create

import (
	"fmt"
	"os"
	"testing"

	_ "github.com/PlakarKorp/integrations/fs/storage"
	"github.com/PlakarKorp/kloset/connectors/storage"
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

	store, err := storage.New(ctx.GetInner(), map[string]string{"location": tmpRepoDirRoot + "/repo"})
	require.NoError(t, err)
	repo := repository.Inexistent(ctx.GetInner(), store)

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

	store, err := storage.New(ctx.GetInner(), map[string]string{"location": tmpRepoDirRoot + "/repo"})
	require.NoError(t, err)
	repo := repository.Inexistent(ctx.GetInner(), store)
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

	store, err := storage.New(ctx.GetInner(), map[string]string{"location": tmpRepoDirRoot + "/repo"})
	require.NoError(t, err)
	repo := repository.Inexistent(ctx.GetInner(), store)
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

func TestCreateParseWeakPassphraseWithKeyfile(t *testing.T) {
	// -weak-passphrase lowers the entropy requirement; combined with a keyfile
	// it parses without prompting.
	ctx := appcontext.NewAppContext()
	defer ctx.Close()
	ctx.KeyFromFile = "weak"

	cmd := &Create{}
	require.NoError(t, cmd.Parse(ctx, []string{"-weak-passphrase"}))
	require.Equal(t, []byte("weak"), cmd.RepositorySecret)
}

func TestCreateParseTooManyParameters(t *testing.T) {
	ctx := appcontext.NewAppContext()
	defer ctx.Close()
	cmd := &Create{}
	err := cmd.Parse(ctx, []string{"-plaintext", "extra-arg"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "too many parameters")
}

func TestCreateParseUnknownHashing(t *testing.T) {
	ctx := appcontext.NewAppContext()
	defer ctx.Close()
	cmd := &Create{}
	err := cmd.Parse(ctx, []string{"-plaintext", "-hashing", "NOTAHASH"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown hashing algorithm")
}

func TestExecuteCmdCreateUnknownHashingConfig(t *testing.T) {
	// Execute looks up the hashing default configuration; an unknown algorithm
	// that slips past Parse (set directly on the struct) makes it fail.
	tmpRepoDirRoot, err := os.MkdirTemp("", "tmp_repo")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(tmpRepoDirRoot) })
	ctx := appcontext.NewAppContext()
	defer ctx.Close()

	store, err := storage.New(ctx.GetInner(), map[string]string{"location": tmpRepoDirRoot + "/repo"})
	require.NoError(t, err)
	repo := repository.Inexistent(ctx.GetInner(), store)

	cmd := &Create{NoEncryption: true, Hashing: "NOTAHASH"}
	status, err := cmd.Execute(ctx, repo)
	require.Error(t, err)
	require.Equal(t, 1, status)
}

func TestExecuteCmdCreateDefaultWithKeyfile(t *testing.T) {
	tmpRepoDirRoot, err := os.MkdirTemp("", "tmp_repo")
	require.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(tmpRepoDirRoot)
	})
	ctx := appcontext.NewAppContext()
	defer ctx.Close()

	store, err := storage.New(ctx.GetInner(), map[string]string{"location": tmpRepoDirRoot + "/repo"})
	require.NoError(t, err)
	repo := repository.Inexistent(ctx.GetInner(), store)
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
