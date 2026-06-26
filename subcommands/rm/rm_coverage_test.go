package rm

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
)

// Parse with neither a filter nor an explicit snapshot is rejected to avoid a
// catastrophic "remove everything".
func TestExecuteCmdRmParseNoFilter(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()
	_ = repo

	cmd := &Rm{}
	err := cmd.Parse(ctx, []string{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "no filter specified")
}

// A filter that matches no snapshot results in a clean no-op (status 0,
// "no snapshots matched").
func TestExecuteCmdRmNoMatch(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	// A bogus id filter matches nothing.
	cmd := &Rm{}
	require.NoError(t, cmd.Parse(ctx, []string{"-apply", "ffffffffffffffff"}))

	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.Contains(t, bufOut.String(), "rm: no snapshots matched the selection")
}

// Dry-run plan against an explicit snapshot prints the plan with no deletions.
func TestExecuteCmdRmDryRunExplicit(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	short := snap.Header.GetIndexShortID()
	cmd := &Rm{}
	require.NoError(t, cmd.Parse(ctx, []string{hex.EncodeToString(short)}))

	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	out := bufOut.String()
	require.Contains(t, out, "rm: would remove these 1 snapshot(s), run with -apply to proceed")
	require.NotContains(t, out, "rm: removal of")
}
