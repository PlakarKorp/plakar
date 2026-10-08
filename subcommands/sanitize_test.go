package subcommands_test

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/PlakarKorp/plakar/subcommands/diag"
	"github.com/PlakarKorp/plakar/subcommands/info"
	"github.com/PlakarKorp/plakar/subcommands/ls"
	"github.com/PlakarKorp/plakar/subcommands/prune"
	"github.com/PlakarKorp/plakar/subcommands/rm"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

// Snapshot metadata comes from whoever wrote the repository and must not
// reach the terminal as raw control sequences.
func TestSnapshotMetadataIsSanitized(t *testing.T) {
	const evil = "evil\x1b[2J\x07"
	const safe = "evil?[2J?"

	bufOut := bytes.NewBuffer(nil)
	repo, ctx := ptesting.GenerateRepository(t, bufOut, bytes.NewBuffer(nil), nil)
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockFile(evil, 0644, "a"),
	}, ptesting.WithName(evil), ptesting.WithTags([]string{evil}))
	defer snap.Close()

	indexID := snap.Header.GetIndexID()
	id := hex.EncodeToString(indexID[:])

	tests := []struct {
		name string
		cmd  subcommands.Subcommand
		args []string
	}{
		{"ls", &ls.Ls{}, []string{"-tags"}},
		{"ls uuid", &ls.Ls{}, []string{"-tags", "-uuid"}},
		{"rm", &rm.Rm{}, []string{"-latest"}},
		{"prune", &prune.Prune{}, []string{"-per-minute=1"}},
		{"info", &info.Info{}, []string{id}},
		{"diag snapshot", &diag.DiagSnapshot{}, []string{id}},
		{"diag vfs dir", &diag.DiagVFS{}, []string{id + ":/"}},
		{"diag vfs file", &diag.DiagVFS{}, []string{id + ":/" + evil}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bufOut.Reset()

			require.NoError(t, tt.cmd.Parse(ctx, tt.args))
			_, err := tt.cmd.Execute(ctx, repo)
			require.NoError(t, err)

			out := bufOut.String()
			require.Contains(t, out, safe)
			require.NotContains(t, out, "\x1b")
			require.NotContains(t, out, "\x07")
		})
	}
}
