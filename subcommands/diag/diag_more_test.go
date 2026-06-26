package diag

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	_ "github.com/PlakarKorp/integration-fs/exporter"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/stretchr/testify/require"
)

// dumpState runs `diag state` (no args) to get the state id, then `diag state
// <id>` and returns the detailed listing (lines like "chunk <mac> : ...").
func dumpState(t *testing.T, ctx *appcontext.AppContext, repo *repository.Repository, bufOut *bytes.Buffer) string {
	t.Helper()

	bufOut.Reset()
	args := []string{"diag", "state"}
	subcommand, _, args := subcommands.Lookup(args)
	require.NoError(t, subcommand.Parse(ctx, args))
	status, err := subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	stateID := strings.Trim(bufOut.String(), "\n")
	require.NotEmpty(t, stateID)

	bufOut.Reset()
	args = []string{"diag", "state", stateID}
	subcommand, _, args = subcommands.Lookup(args)
	require.NoError(t, subcommand.Parse(ctx, args))
	status, err = subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	return bufOut.String()
}

// macForPrefix returns the MAC of the first state line starting with prefix
// (e.g. "chunk ", "object ").
func macForPrefix(t *testing.T, stateOutput, prefix string) string {
	t.Helper()
	for _, line := range strings.Split(stateOutput, "\n") {
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		var mac, packfile []byte
		var offset, length int
		format := prefix + "%x : packfile %x, offset %d, length %d"
		if _, err := fmt.Sscanf(line, format, &mac, &packfile, &offset, &length); err == nil && len(mac) == 32 {
			return hex.EncodeToString(mac)
		}
	}
	t.Fatalf("no %q entry found in state output", prefix)
	return ""
}

func TestExecuteCmdDiagBlobSnapshot(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	stateOut := dumpState(t, ctx, repo, bufOut)
	snapMAC := macForPrefix(t, stateOut, "snapshot ")

	bufOut.Reset()
	args := []string{"diag", "blob", "snapshot", snapMAC}
	subcommand, _, args := subcommands.Lookup(args)
	require.NoError(t, subcommand.Parse(ctx, args))
	require.NotNil(t, subcommand)

	status, err := subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	require.NotEmpty(t, strings.TrimSpace(bufOut.String()))
}

func TestExecuteCmdDiagBlobObject(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	stateOut := dumpState(t, ctx, repo, bufOut)
	objectMAC := macForPrefix(t, stateOut, "object ")

	bufOut.Reset()
	args := []string{"diag", "blob", "object", objectMAC}
	subcommand, _, args := subcommands.Lookup(args)
	require.NoError(t, subcommand.Parse(ctx, args))

	status, err := subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	require.NotEmpty(t, strings.TrimSpace(bufOut.String()))
}

func TestExecuteCmdDiagBlobParseAndExecuteErrors(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	// Missing arguments -> Parse error.
	args := []string{"diag", "blob", "chunk"}
	subcommand, _, args := subcommands.Lookup(args)
	require.Error(t, subcommand.Parse(ctx, args))

	// Unknown blob type -> Execute error.
	args = []string{"diag", "blob", "not-a-type", strings.Repeat("00", 32)}
	subcommand, _, args = subcommands.Lookup(args)
	require.NoError(t, subcommand.Parse(ctx, args))
	status, err := subcommand.Execute(ctx, repo)
	require.Error(t, err)
	require.Equal(t, 1, status)

	// Bad hex mac -> Execute error.
	args = []string{"diag", "blob", "chunk", "zz"}
	subcommand, _, args = subcommands.Lookup(args)
	require.NoError(t, subcommand.Parse(ctx, args))
	status, err = subcommand.Execute(ctx, repo)
	require.Error(t, err)
	require.Equal(t, 1, status)

	// Wrong mac length -> Execute error.
	args = []string{"diag", "blob", "chunk", "00ff"}
	subcommand, _, args = subcommands.Lookup(args)
	require.NoError(t, subcommand.Parse(ctx, args))
	status, err = subcommand.Execute(ctx, repo)
	require.Error(t, err)
	require.Equal(t, 1, status)
}

func TestExecuteCmdDiagBlobSearch(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	stateOut := dumpState(t, ctx, repo, bufOut)
	objectMAC := macForPrefix(t, stateOut, "object ")

	bufOut.Reset()
	args := []string{"diag", "blobsearch", objectMAC}
	subcommand, _, args := subcommands.Lookup(args)
	require.NoError(t, subcommand.Parse(ctx, args))
	require.NotNil(t, subcommand)

	status, err := subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	output := bufOut.String()
	require.Contains(t, output, "Warning this command is slow and expensive")
	require.Contains(t, output, "Found candidate")
	require.Contains(t, output, "object: ")
}

func TestExecuteCmdDiagBlobSearchErrors(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	// Missing argument -> Parse error.
	args := []string{"diag", "blobsearch"}
	subcommand, _, args := subcommands.Lookup(args)
	require.Error(t, subcommand.Parse(ctx, args))

	// Wrong-length hash -> Execute error.
	bufOut.Reset()
	args = []string{"diag", "blobsearch", "deadbeef"}
	subcommand, _, args = subcommands.Lookup(args)
	require.NoError(t, subcommand.Parse(ctx, args))
	status, err := subcommand.Execute(ctx, repo)
	require.Error(t, err)
	require.Equal(t, 1, status)

	// Right length but invalid hex -> Execute error.
	bufOut.Reset()
	args = []string{"diag", "blobsearch", strings.Repeat("zz", 32)}
	subcommand, _, args = subcommands.Lookup(args)
	require.NoError(t, subcommand.Parse(ctx, args))
	status, err = subcommand.Execute(ctx, repo)
	require.Error(t, err)
	require.Equal(t, 1, status)
}

func TestExecuteCmdDiagDirPack(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	indexID := snap.Header.GetIndexID()
	args := []string{"diag", "dirpack", hex.EncodeToString(indexID[:])}
	subcommand, _, args := subcommands.Lookup(args)
	require.NoError(t, subcommand.Parse(ctx, args))
	require.NotNil(t, subcommand)

	status, err := subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	require.Contains(t, bufOut.String(), "vfs-entry ")
}

func TestExecuteCmdDiagDirPackParseError(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	_, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	args := []string{"diag", "dirpack"}
	subcommand, _, args := subcommands.Lookup(args)
	require.Error(t, subcommand.Parse(ctx, args))
}

func TestExecuteCmdDiagDirPackBadSnapshot(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	args := []string{"diag", "dirpack", strings.Repeat("00", 32)}
	subcommand, _, args := subcommands.Lookup(args)
	require.NoError(t, subcommand.Parse(ctx, args))
	status, err := subcommand.Execute(ctx, repo)
	require.Error(t, err)
	require.Equal(t, 1, status)
}

func TestExecuteCmdDiagRepository(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	// Default action: `diag` with no subcommand maps to DiagRepository.
	args := []string{"diag"}
	subcommand, _, args := subcommands.Lookup(args)
	require.NoError(t, subcommand.Parse(ctx, args))
	require.NotNil(t, subcommand)

	status, err := subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	output := bufOut.String()
	require.Contains(t, output, "Version:")
	require.Contains(t, output, "RepositoryID:")
	require.Contains(t, output, "Packfile:")
	require.Contains(t, output, "Chunking:")
	require.Contains(t, output, "Hashing:")
	require.Contains(t, output, "Snapshots: 1")
	require.Contains(t, output, "Size:")
}
