package sync

import (
	"bytes"
	"encoding/hex"
	"os"
	"testing"

	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/config"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

func init() {
	os.Setenv("TZ", "UTC")
}

var mockFiles = []ptesting.MockFile{
	ptesting.NewMockDir("subdir"),
	ptesting.NewMockDir("another_subdir"),
	ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
	ptesting.NewMockFile("subdir/foo.txt", 0644, "hello foo"),
	ptesting.NewMockFile("subdir/to_exclude", 0644, "*/subdir/to_exclude\n"),
	ptesting.NewMockFile("another_subdir/bar.txt", 0644, "hello bar"),
}

// syncFixture is the equivalent of what main.go sets up before dispatching to
// the sync subcommand: a local repository (the one the command operates on), a
// peer repository, an in-process cached and a configuration through which the
// peer can be resolved.
type syncFixture struct {
	localRepo *repository.Repository
	localCtx  *appcontext.AppContext
	peerRepo  *repository.Repository

	// what to pass as the REPOSITORY argument of the sync command
	peerArg string

	output *bytes.Buffer
}

func setupSync(t *testing.T, localPassphrase, peerPassphrase []byte) *syncFixture {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	var localPass, peerPass *[]byte
	if localPassphrase != nil {
		localPass = &localPassphrase
	}
	if peerPassphrase != nil {
		peerPass = &peerPassphrase
	}

	localRepo, localCtx := ptesting.GenerateRepository(t, bufOut, bufErr, localPass)
	peerRepo, _ := ptesting.GenerateRepository(t, bufOut, bufErr, peerPass)

	// sync taps into cached to rebuild states, run one in-process on the
	// local context's cache directory.
	ptesting.StartCached(t, localCtx)

	// main.go sets the resolved store config of the repository being
	// operated on; the state refresher relies on it.
	localCtx.StoreConfig = map[string]string{"location": localRepo.Root()}

	fixture := &syncFixture{
		localRepo: localRepo,
		localCtx:  localCtx,
		peerRepo:  peerRepo,
		peerArg:   peerRepo.Root(),
		output:    bufOut,
	}

	localCtx.Config = config.NewConfig()
	if peerPassphrase != nil {
		// an encrypted peer can only provide its passphrase through the
		// configuration (or interactively), address it as @peer.
		localCtx.Config.Repositories["peer"] = map[string]string{
			"location":   peerRepo.Root(),
			"passphrase": string(peerPassphrase),
		}
		fixture.peerArg = "@peer"
	}

	return fixture
}

func snapshotIDs(t *testing.T, repo *repository.Repository) map[objects.MAC]struct{} {
	require.NoError(t, repo.RebuildState())

	ids := make(map[objects.MAC]struct{})
	for id, err := range repo.ListSnapshots() {
		require.NoError(t, err)
		ids[id] = struct{}{}
	}
	return ids
}

func runSync(t *testing.T, fixture *syncFixture, args []string) {
	subcommand := &Sync{}
	err := subcommand.Parse(fixture.localCtx, args)
	require.NoError(t, err)

	status, err := subcommand.Execute(fixture.localCtx, fixture.localRepo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
}

func testSyncDirection(t *testing.T, direction string, localPassphrase, peerPassphrase []byte) {
	fixture := setupSync(t, localPassphrase, peerPassphrase)

	wantSynchronized := 1
	var localSnap, peerSnap *objects.MAC
	switch direction {
	case "to":
		snap := ptesting.GenerateSnapshot(t, fixture.localRepo, mockFiles)
		defer snap.Close()
		localSnap = &snap.Header.Identifier
	case "from":
		snap := ptesting.GenerateSnapshot(t, fixture.peerRepo, mockFiles)
		defer snap.Close()
		peerSnap = &snap.Header.Identifier
	case "with":
		lsnap := ptesting.GenerateSnapshot(t, fixture.localRepo, mockFiles)
		defer lsnap.Close()
		localSnap = &lsnap.Header.Identifier
		psnap := ptesting.GenerateSnapshot(t, fixture.peerRepo, mockFiles)
		defer psnap.Close()
		peerSnap = &psnap.Header.Identifier
		wantSynchronized = 2
	}

	runSync(t, fixture, []string{direction, fixture.peerArg})

	// whatever the direction, both ends must now hold every snapshot
	localIDs := snapshotIDs(t, fixture.localRepo)
	peerIDs := snapshotIDs(t, fixture.peerRepo)
	require.Len(t, localIDs, wantSynchronized)
	require.Len(t, peerIDs, wantSynchronized)
	if localSnap != nil {
		require.Contains(t, localIDs, *localSnap)
		require.Contains(t, peerIDs, *localSnap)
	}
	if peerSnap != nil {
		require.Contains(t, localIDs, *peerSnap)
		require.Contains(t, peerIDs, *peerSnap)
	}

	require.Regexp(t,
		`info: sync: synchronization .*completed: `+string(rune('0'+wantSynchronized))+` snapshots synchronized`,
		fixture.output.String())
}

func TestExecuteCmdSync(t *testing.T) {
	localPassphrase := []byte("aZeRtY123456$#@!@")
	// different from the local one on purpose: rebuilding the peer's state
	// with the local crypto material must not work.
	peerPassphrase := []byte("QsDfG654321&^%*%!")

	combinations := []struct {
		name      string
		localPass []byte
		peerPass  []byte
	}{
		{"plain_plain", nil, nil},
		{"plain_encrypted", nil, peerPassphrase},
		{"encrypted_plain", localPassphrase, nil},
		{"encrypted_encrypted", localPassphrase, peerPassphrase},
	}

	for _, direction := range []string{"to", "from", "with"} {
		for _, combo := range combinations {
			t.Run(direction+"_"+combo.name, func(t *testing.T) {
				testSyncDirection(t, direction, combo.localPass, combo.peerPass)
			})
		}
	}
}

func TestExecuteCmdSyncSnapshotID(t *testing.T) {
	fixture := setupSync(t, nil, nil)

	snap := ptesting.GenerateSnapshot(t, fixture.localRepo, mockFiles)
	defer snap.Close()

	indexId := snap.Header.GetIndexID()
	runSync(t, fixture, []string{hex.EncodeToString(indexId[:]), "to", fixture.peerArg})

	peerIDs := snapshotIDs(t, fixture.peerRepo)
	require.Contains(t, peerIDs, snap.Header.Identifier)
}
