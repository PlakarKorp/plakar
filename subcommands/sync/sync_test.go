package sync

import (
	"bytes"
	"encoding/hex"
	"os"
	"strings"
	"testing"

	_ "github.com/PlakarKorp/integration-fs/exporter"
	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/config"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/PlakarKorp/plakar/utils"
	"github.com/stretchr/testify/require"
)

func init() {
	os.Setenv("TZ", "UTC")
}

// NOTE ON Execute COVERAGE:
// Sync.Execute calls cached.RebuildStateFromStore, which spawns a "cached"
// daemon by re-executing os.Executable() with the "cached" argument. Under
// `go test` the executable is the test binary, which does not implement the
// cached subcommand, so the helper retries the connection up to 1000 times,
// spawning a process each round, and effectively hangs. This is why the
// repo's pre-existing Execute tests were disabled (renamed with a leading
// underscore). The hermetic, non-daemon surface of this package is Parse,
// which is what these tests exercise thoroughly (argument validation, the
// snapshot/filter forms, and all three encrypted-peer passphrase-derivation
// branches including the wrong-passphrase failures).

func generateSnapshot(t *testing.T, bufOut *bytes.Buffer, bufErr *bytes.Buffer) (*repository.Repository, *snapshot.Snapshot, *appcontext.AppContext) {
	stateRefresher = func(*appcontext.AppContext, *repository.Repository) func(objects.MAC, bool) error {
		return nil
	}

	repo, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)
	// sync's Parse needs a non-nil config to resolve peer repository locations.
	ctx.Config = config.NewConfig()
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockDir("another_subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
		ptesting.NewMockFile("subdir/foo.txt", 0644, "hello foo"),
		ptesting.NewMockFile("subdir/to_exclude", 0644, "*/subdir/to_exclude\n"),
		ptesting.NewMockFile("another_subdir/bar.txt", 0644, "hello bar"),
	})
	return repo, snap, ctx
}

// ---------------------------------------------------------------------------
// Parse validation
// ---------------------------------------------------------------------------

func TestSyncParseTooManyArguments(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	_, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	cmd := &Sync{}
	err := cmd.Parse(ctx, []string{"a", "to", "b", "c"})
	require.EqualError(t, err, "Too many arguments")
}

func TestSyncParseWrongArgCount(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	_, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	t.Run("zero args", func(t *testing.T) {
		cmd := &Sync{}
		err := cmd.Parse(ctx, []string{})
		require.EqualError(t, err, "usage: sync [SNAPSHOT] to|from|with REPOSITORY")
	})
	t.Run("one arg", func(t *testing.T) {
		cmd := &Sync{}
		err := cmd.Parse(ctx, []string{"to"})
		require.EqualError(t, err, "usage: sync [SNAPSHOT] to|from|with REPOSITORY")
	})
}

func TestSyncParseInvalidDirection(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	_, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	cmd := &Sync{}
	err := cmd.Parse(ctx, []string{"sideways", "somewhere"})
	require.EqualError(t, err, "invalid direction, must be to, from or with")
}

func TestSyncParseUnknownPeer(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	_, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	cmd := &Sync{}
	err := cmd.Parse(ctx, []string{"to", "@doesnotexist"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "peer store")
	require.Contains(t, err.Error(), "could not resolve repository")
}

// Three-arg form: SNAPSHOT direction REPOSITORY, with a valid (plaintext) peer.
func TestSyncParseSnapshotForm(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	_, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	peerRepo, _ := ptesting.GenerateRepository(t, bufOut, bufErr, nil)

	indexId := snap.Header.GetIndexID()
	cmd := &Sync{}
	err := cmd.Parse(ctx, []string{hex.EncodeToString(indexId[:]), "to", peerRepo.Root()})
	require.NoError(t, err)
	require.Equal(t, "to", cmd.Direction)
	require.Equal(t, peerRepo.Root(), cmd.PeerRepositoryLocation)
	require.Equal(t, []string{hex.EncodeToString(indexId[:])}, cmd.SrcLocateOptions.Filters.IDs)
}

// Two-arg form: direction REPOSITORY (no snapshot, filters keep defaults).
func TestSyncParseFilterForm(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	_, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	peerRepo, _ := ptesting.GenerateRepository(t, bufOut, bufErr, nil)

	cmd := &Sync{}
	err := cmd.Parse(ctx, []string{"with", peerRepo.Root()})
	require.NoError(t, err)
	require.Equal(t, "with", cmd.Direction)
	require.Empty(t, cmd.SrcLocateOptions.Filters.IDs)
}

// ---------------------------------------------------------------------------
// Encrypted peer: passphrase derivation paths in Parse
// ---------------------------------------------------------------------------

// encryptedPeer creates an encrypted peer repository and registers it in the
// caller's config under @peerRepo, then returns the config-file path so the
// caller can tweak passphrase / passphrase_cmd before Parse.
func encryptedPeer(t *testing.T, ctx *appcontext.AppContext, bufOut, bufErr *bytes.Buffer, passphrase string) *repository.Repository {
	pp := []byte(passphrase)
	peerRepo, _ := ptesting.GenerateRepository(t, bufOut, bufErr, &pp)

	opt_configfile := strings.TrimPrefix(peerRepo.Root(), "fs://")
	cfg, err := utils.LoadConfig(opt_configfile)
	require.NoError(t, err)
	ctx.Config = cfg
	ctx.Config.Repositories["peerRepo"] = map[string]string{
		"location": peerRepo.Root(),
	}
	err = utils.SaveConfig(opt_configfile, ctx.Config)
	require.NoError(t, err)
	return peerRepo
}

func TestSyncParseEncryptedPeerPassphrase(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	_, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	passphrase := "aZeRtY123456$#@!@"
	encryptedPeer(t, ctx, bufOut, bufErr, passphrase)
	ctx.Config.Repositories["peerRepo"]["passphrase"] = passphrase

	indexId := snap.Header.GetIndexID()
	cmd := &Sync{}
	err := cmd.Parse(ctx, []string{hex.EncodeToString(indexId[:]), "with", "@peerRepo"})
	require.NoError(t, err)
	require.NotEmpty(t, cmd.PeerRepositorySecret)
}

func TestSyncParseEncryptedPeerWrongPassphrase(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	_, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	encryptedPeer(t, ctx, bufOut, bufErr, "correct-horse-battery")
	ctx.Config.Repositories["peerRepo"]["passphrase"] = "totally-wrong"

	indexId := snap.Header.GetIndexID()
	cmd := &Sync{}
	err := cmd.Parse(ctx, []string{hex.EncodeToString(indexId[:]), "with", "@peerRepo"})
	require.EqualError(t, err, "invalid passphrase")
}

func TestSyncParseEncryptedPeerPassphraseCmd(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	_, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	// Plain alphanumeric: the passphrase_cmd is run through /bin/sh -c, so shell
	// metacharacters would be expanded and corrupt the value.
	passphrase := "correcthorsebatterystaple"
	encryptedPeer(t, ctx, bufOut, bufErr, passphrase)
	ctx.Config.Repositories["peerRepo"]["passphrase_cmd"] = "echo " + passphrase

	indexId := snap.Header.GetIndexID()
	cmd := &Sync{}
	err := cmd.Parse(ctx, []string{hex.EncodeToString(indexId[:]), "with", "@peerRepo"})
	require.NoError(t, err)
	require.NotEmpty(t, cmd.PeerRepositorySecret)
}

func TestSyncParseEncryptedPeerPassphraseCmdWrong(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	_, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	encryptedPeer(t, ctx, bufOut, bufErr, "therightone")
	ctx.Config.Repositories["peerRepo"]["passphrase_cmd"] = "echo nope"

	indexId := snap.Header.GetIndexID()
	cmd := &Sync{}
	err := cmd.Parse(ctx, []string{hex.EncodeToString(indexId[:]), "with", "@peerRepo"})
	require.EqualError(t, err, "invalid passphrase")
}

// A passphrase_cmd that fails (non-zero exit / no output) surfaces a read error.
func TestSyncParseEncryptedPeerPassphraseCmdFails(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	_, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	encryptedPeer(t, ctx, bufOut, bufErr, "whatever")
	// emits zero lines -> GetPassphraseFromCommand returns "too many lines" error
	ctx.Config.Repositories["peerRepo"]["passphrase_cmd"] = "true"

	indexId := snap.Header.GetIndexID()
	cmd := &Sync{}
	err := cmd.Parse(ctx, []string{hex.EncodeToString(indexId[:]), "with", "@peerRepo"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to read passphrase from command")
}
