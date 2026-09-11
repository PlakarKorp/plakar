package sync

import (
	"testing"

	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

// TestCov80SyncParseInvalidPassphrase configures an encrypted peer with the
// wrong passphrase in the config. Parse derives the key and the canary
// verification fails, so it must reject with "invalid passphrase". This drives
// the VerifyCanary failure branch of the "passphrase" path in Parse.
func TestCov80SyncParseInvalidPassphrase(t *testing.T) {
	peerPass := []byte("the-real-passphrase")
	fixture := setupSync(t, nil, peerPass)

	// Override the configured peer entry with an incorrect passphrase.
	fixture.localCtx.Config.Repositories["peer"] = map[string]string{
		"location":   fixture.peerRepo.Root(),
		"passphrase": "wrong-passphrase",
	}

	err := (&Sync{}).Parse(fixture.localCtx, []string{"to", "@peer"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid passphrase")
}

// TestCov80SyncParsePassphraseCmd configures an encrypted peer that supplies its
// passphrase via the "passphrase_cmd" config key. Parse runs the command,
// derives the key, and the canary verification succeeds. This drives the
// passphrase_cmd branch of Parse that the standard passphrase-in-config tests
// do not reach.
func TestCov80SyncParsePassphraseCmd(t *testing.T) {
	peerPass := []byte("cmd-supplied-pass")
	fixture := setupSync(t, nil, peerPass)

	fixture.localCtx.Config.Repositories["peer"] = map[string]string{
		"location":       fixture.peerRepo.Root(),
		"passphrase_cmd": "printf %s cmd-supplied-pass",
	}

	err := (&Sync{}).Parse(fixture.localCtx, []string{"to", "@peer"})
	require.NoError(t, err)
}

// TestCov80SyncParsePassphraseCmdInvalid drives the passphrase_cmd branch where
// the command yields the wrong passphrase: the canary check fails and Parse
// returns "invalid passphrase".
func TestCov80SyncParsePassphraseCmdInvalid(t *testing.T) {
	peerPass := []byte("cmd-supplied-pass")
	fixture := setupSync(t, nil, peerPass)

	fixture.localCtx.Config.Repositories["peer"] = map[string]string{
		"location":       fixture.peerRepo.Root(),
		"passphrase_cmd": "echo -n totally-wrong",
	}

	err := (&Sync{}).Parse(fixture.localCtx, []string{"to", "@peer"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid passphrase")
}

// TestCov80SyncExecuteFromNoSnapshot drives Execute with direction "from"
// against an empty peer: the source/dest swap branch runs and LocateSnapshotIDs
// returns nothing, so Execute short-circuits with status 0 and an
// informational log. This complements the existing "to" no-match test.
func TestCov80SyncExecuteFromNoSnapshot(t *testing.T) {
	fixture := setupSync(t, nil, nil)

	cmd := &Sync{}
	require.NoError(t, cmd.Parse(fixture.localCtx, []string{"from", fixture.peerArg}))
	status, err := cmd.Execute(fixture.localCtx, fixture.localRepo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.Contains(t, fixture.output.String(), "No matching snapshot found")
}

// TestCov80SyncExecuteFromTransfersSnapshot drives a real "from" sync of a
// single snapshot living on the peer into the local repo. This exercises the
// from-direction context swap plus the synchronize() path in the from
// direction, which the "to"-oriented coverage tests do not cover.
func TestCov80SyncExecuteFromTransfersSnapshot(t *testing.T) {
	fixture := setupSync(t, nil, nil)
	snap := ptesting.GenerateSnapshot(t, fixture.peerRepo, mockFiles)
	defer snap.Close()

	cmd := &Sync{}
	require.NoError(t, cmd.Parse(fixture.localCtx, []string{"from", fixture.peerArg}))
	status, err := cmd.Execute(fixture.localCtx, fixture.localRepo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	localIDs := snapshotIDs(t, fixture.localRepo)
	require.Contains(t, localIDs, snap.Header.Identifier)
}
