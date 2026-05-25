package sync

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"testing"

	_ "github.com/PlakarKorp/integration-fs/exporter"
	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/plakar/appcontext"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/PlakarKorp/plakar/utils"
	"github.com/stretchr/testify/require"
)

func init() {
	os.Setenv("TZ", "UTC")
}

func generateSnapshot(t *testing.T, bufOut *bytes.Buffer, bufErr *bytes.Buffer) (*repository.Repository, *snapshot.Snapshot, *appcontext.AppContext) {
	// Disable the stateRefresher hook so RebuildStateFromStateFile during the
	// sync flow becomes a no-op. The peer-side RebuildStateFromStore call is
	// handled separately by the in-process fake cached server.
	stateRefresher = func(*appcontext.AppContext, *repository.Repository) func(objects.MAC, bool) error {
		return nil
	}

	repo, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)
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

func TestExecuteCmdSyncTo(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	localRepo, snap, lctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	// Stand up an in-process fake cached server so the peer-side
	// cached.RebuildStateFromStore call in Sync.Execute (sync.go:194) doesn't
	// try to fork-exec the test binary as a cached daemon.
	cachedSrv := ptesting.StartFakeCachedServer(t, lctx)
	defer cachedSrv.Close()

	peerRepo, _ := ptesting.GenerateRepository(t, bufOut, bufErr, nil)

	indexId := snap.Header.GetIndexID()
	args := []string{hex.EncodeToString(indexId[:]), "to", peerRepo.Root()}

	subcommand := &Sync{}
	err := subcommand.Parse(lctx, args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	status, err := subcommand.Execute(lctx, localRepo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	output := bufOut.String()
	require.Contains(t, strings.Trim(output, "\n"), fmt.Sprintf("info: sync: synchronization from %s to %s completed: 1 snapshots synchronized", localRepo.Origin(), peerRepo.Origin()))
}

func TestExecuteCmdSyncWith(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	localRepo, snap, lctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	cachedSrv := ptesting.StartFakeCachedServer(t, lctx)
	defer cachedSrv.Close()

	peerRepo, _ := ptesting.GenerateRepository(t, bufOut, bufErr, nil)

	indexId := snap.Header.GetIndexID()
	args := []string{hex.EncodeToString(indexId[:]), "with", peerRepo.Root()}

	subcommand := &Sync{}
	err := subcommand.Parse(lctx, args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	status, err := subcommand.Execute(lctx, localRepo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	output := bufOut.String()
	require.Contains(t, strings.Trim(output, "\n"), fmt.Sprintf("info: sync: synchronization between %s and %s completed: 1 snapshots synchronized", localRepo.Origin(), peerRepo.Origin()))
}

func TestExecuteCmdSyncWithEncryption(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	localRepo, snap, lctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	cachedSrv := ptesting.StartFakeCachedServer(t, lctx)
	defer cachedSrv.Close()

	passphrase := []byte("aZeRtY123456$#@!@")
	peerRepo, _ := ptesting.GenerateRepository(t, bufOut, bufErr, &passphrase)

	// Recreate the config so the passphrase is stored alongside the peer repo.
	opt_configfile := strings.TrimPrefix(peerRepo.Root(), "fs://")

	cfg, err := utils.LoadConfig(opt_configfile)
	require.NoError(t, err)
	lctx.Config = cfg
	lctx.Config.Repositories["peerRepo"] = make(map[string]string)
	lctx.Config.Repositories["peerRepo"]["passphrase"] = string(passphrase)
	lctx.Config.Repositories["peerRepo"]["location"] = peerRepo.Root()
	err = utils.SaveConfig(opt_configfile, lctx.Config)
	require.NoError(t, err)

	indexId := snap.Header.GetIndexID()
	args := []string{hex.EncodeToString(indexId[:]), "with", "@peerRepo"}

	subcommand := &Sync{}
	err = subcommand.Parse(lctx, args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	status, err := subcommand.Execute(lctx, localRepo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	output := bufOut.String()
	require.Contains(t, strings.Trim(output, "\n"), fmt.Sprintf("info: sync: synchronization between %s and %s completed: 1 snapshots synchronized", localRepo.Origin(), peerRepo.Origin()))
}

func TestParseRejectsNoArgs(t *testing.T) {
	_, _, ctx := generateSnapshot(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil))
	subcommand := &Sync{}
	err := subcommand.Parse(ctx, []string{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "usage")
}

func TestParseRejectsInvalidDirection(t *testing.T) {
	_, _, ctx := generateSnapshot(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil))
	subcommand := &Sync{}
	err := subcommand.Parse(ctx, []string{"sideways", "fs:///tmp"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid direction")
}

func TestParseRejectsTooManyArgs(t *testing.T) {
	_, _, ctx := generateSnapshot(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil))
	subcommand := &Sync{}
	err := subcommand.Parse(ctx, []string{"a", "b", "c", "d"})
	require.Error(t, err)
}

func TestParseRejectsUnknownPeer(t *testing.T) {
	_, _, ctx := generateSnapshot(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil))
	// Parse looks up @-aliases through ctx.Config; generateSnapshot leaves it
	// nil. Initialize with an empty config so the lookup returns "not found"
	// instead of NPE.
	ctx.ConfigDir = t.TempDir()
	require.NoError(t, ctx.ReloadConfig())
	subcommand := &Sync{}
	err := subcommand.Parse(ctx, []string{"to", "@nope-this-is-not-configured"})
	require.Error(t, err)
}

func TestExecuteCmdSyncFrom(t *testing.T) {
	// Reverse direction: pull a snapshot from the peer into the local repo.
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	localRepo, _, lctx := generateSnapshot(t, bufOut, bufErr)

	cachedSrv := ptesting.StartFakeCachedServer(t, lctx)
	defer cachedSrv.Close()

	peerRepo, peerCtx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	peerSnap := ptesting.GenerateSnapshot(t, peerRepo, []ptesting.MockFile{
		ptesting.NewMockFile("hello.txt", 0644, "from peer"),
	})
	defer peerSnap.Close()
	_ = peerCtx

	indexId := peerSnap.Header.GetIndexID()
	args := []string{hex.EncodeToString(indexId[:]), "from", peerRepo.Root()}

	subcommand := &Sync{}
	err := subcommand.Parse(lctx, args)
	require.NoError(t, err)

	status, err := subcommand.Execute(lctx, localRepo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
}
