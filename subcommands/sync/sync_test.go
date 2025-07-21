package sync

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/plakar/appcontext"
	_ "github.com/PlakarKorp/plakar/connectors/fs/exporter"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/PlakarKorp/plakar/utils"
	"github.com/stretchr/testify/require"
)

func init() {
	os.Setenv("TZ", "UTC")
}

func generateSnapshot(t *testing.T, bufOut *bytes.Buffer, bufErr *bytes.Buffer) (*repository.Repository, *snapshot.Snapshot, *appcontext.AppContext) {
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

	peerRepo, _ := ptesting.GenerateRepository(t, bufOut, bufErr, nil)

	indexId := snap.Header.GetIndexID()
	args := []string{fmt.Sprintf("%s", hex.EncodeToString(indexId[:])), "to", peerRepo.Location()}

	subcommand := &Sync{}
	err := subcommand.Parse(lctx, args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	status, err := subcommand.Execute(lctx, localRepo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	// output should look like this
	// 2025-03-26T21:17:28Z info: sync: synchronization from /tmp/tmp_repo1957539148/repo to /tmp/tmp_repo2470692775/repo completed: 1 snapshots synchronized
	output := bufOut.String()
	require.Contains(t, strings.Trim(output, "\n"), fmt.Sprintf("info: sync: synchronization from %s to %s completed: 1 snapshots synchronized", localRepo.Location(), peerRepo.Location()))
}

func TestExecuteCmdSyncWith(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	localRepo, snap, lctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	peerRepo, _ := ptesting.GenerateRepository(t, bufOut, bufErr, nil)

	indexId := snap.Header.GetIndexID()
	args := []string{fmt.Sprintf("%s", hex.EncodeToString(indexId[:])), "with", peerRepo.Location()}

	subcommand := &Sync{}
	err := subcommand.Parse(lctx, args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	status, err := subcommand.Execute(lctx, localRepo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	// output should look like this
	// 2025-03-26T21:28:23Z info: sync: synchronization between /tmp/tmp_repo3863826583/repo and /tmp/tmp_repo327669581/repo completed: 1 snapshots synchronized
	output := bufOut.String()
	require.Contains(t, strings.Trim(output, "\n"), fmt.Sprintf("info: sync: synchronization between %s and %s completed: 1 snapshots synchronized", localRepo.Location(), peerRepo.Location()))
}

func TestExecuteCmdSyncWithEncryption(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	localRepo, snap, lctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	passphrase := []byte("aZeRtY123456$#@!@")
	peerRepo, _ := ptesting.GenerateRepository(t, bufOut, bufErr, &passphrase)

	// need to recreate configuration to store passphrase on peer repo
	opt_configfile := filepath.Join(strings.TrimPrefix(peerRepo.Location(), "fs://"))

	cfg, err := utils.LoadConfig(opt_configfile)
	require.NoError(t, err)
	lctx.Config = cfg
	lctx.Config.Repositories["peerRepo"] = make(map[string]string)
	lctx.Config.Repositories["peerRepo"]["passphrase"] = string(passphrase)
	lctx.Config.Repositories["peerRepo"]["location"] = string(peerRepo.Location())
	err = utils.SaveConfig(opt_configfile, lctx.Config)
	require.NoError(t, err)

	indexId := snap.Header.GetIndexID()
	args := []string{fmt.Sprintf("%s", hex.EncodeToString(indexId[:])), "with", "@peerRepo"}

	subcommand := &Sync{}
	err = subcommand.Parse(lctx, args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	status, err := subcommand.Execute(lctx, localRepo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	// output should look like this
	// 2025-03-26T21:28:23Z info: sync: synchronization between /tmp/tmp_repo3863826583/repo and /tmp/tmp_repo327669581/repo completed: 1 snapshots synchronized
	output := bufOut.String()
	require.Contains(t, strings.Trim(output, "\n"), fmt.Sprintf("info: sync: synchronization between %s and %s completed: 1 snapshots synchronized", localRepo.Location(), peerRepo.Location()))
}
