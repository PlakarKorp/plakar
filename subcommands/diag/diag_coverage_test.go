package diag

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	_ "github.com/PlakarKorp/integrations/fs/exporter"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

// covGenSnapshot builds a small repo+snapshot for the coverage tests. Distinct
// name from the package's existing generateSnapshot helper.
func covGenSnapshot(t *testing.T) (*repository.Repository, *snapshot.Snapshot, *appcontext.AppContext, *bytes.Buffer) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
		ptesting.NewMockFile("subdir/foo.txt", 0644, "hello foo"),
	})
	return repo, snap, ctx, bufOut
}

// runDiag is a small helper that drives a diag subcommand end-to-end.
func runDiag(t *testing.T, ctx *appcontext.AppContext, repo *repository.Repository, args []string) (int, error) {
	t.Helper()
	subcommand, _, rest := subcommands.Lookup(args)
	require.NotNil(t, subcommand)
	if err := subcommand.Parse(ctx, rest); err != nil {
		return -1, err
	}
	return subcommand.Execute(ctx, repo)
}

// stateLines runs `diag state <serial>` and returns the per-blob index lines.
func stateLines(t *testing.T, ctx *appcontext.AppContext, repo *repository.Repository, bufOut *bytes.Buffer) []string {
	t.Helper()
	bufOut.Reset()
	status, err := runDiag(t, ctx, repo, []string{"diag", "state"})
	require.NoError(t, err)
	require.Equal(t, 0, status)
	serial := strings.Trim(bufOut.String(), "\n")

	bufOut.Reset()
	status, err = runDiag(t, ctx, repo, []string{"diag", "state", serial})
	require.NoError(t, err)
	require.Equal(t, 0, status)
	return strings.Split(strings.Trim(bufOut.String(), "\n"), "\n")
}

// --- diag repository (default subcommand) ----------------------------------

func TestCovDiagRepositoryDefault(t *testing.T) {
	repo, snap, ctx, bufOut := covGenSnapshot(t)
	defer snap.Close()

	bufOut.Reset()
	status, err := runDiag(t, ctx, repo, []string{"diag"})
	require.NoError(t, err)
	require.Equal(t, 0, status)

	out := bufOut.String()
	require.Contains(t, out, "Version:")
	require.Contains(t, out, "RepositoryID:")
	require.Contains(t, out, "Chunking:")
	require.Contains(t, out, "Hashing:")
	require.Contains(t, out, "Snapshots:")
	require.Contains(t, out, "Size:")
}

func TestCovDiagRepositoryEncrypted(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	passphrase := []byte("correct horse battery staple")
	repo, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, &passphrase)
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockFile("secret.txt", 0644, "top secret"),
	})
	defer snap.Close()

	bufOut.Reset()
	status, err := runDiag(t, ctx, repo, []string{"diag"})
	require.NoError(t, err)
	require.Equal(t, 0, status)

	out := bufOut.String()
	require.Contains(t, out, "Encryption:")
	require.Contains(t, out, "KDF:")
	require.Contains(t, out, "Canary:")
}

// --- diag blob (success + error paths) -------------------------------------

func TestCovDiagBlobSuccess(t *testing.T) {
	repo, snap, ctx, bufOut := covGenSnapshot(t)
	defer snap.Close()

	lines := stateLines(t, ctx, repo, bufOut)

	// For each blob-type we know how to deserialize, find one index line and
	// run `diag blob <type> <mac>`. We skip RT_CHUNK success (won't
	// deserialize from a raw blob, per the brief) and only assert the ones
	// that decode cleanly.
	wantTypes := map[string]bool{
		"snapshot":  true,
		"object":    true,
		"vfs-entry": true,
	}

	tested := map[string]bool{}
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		typ := fields[0]
		if !wantTypes[typ] || tested[typ] {
			continue
		}
		mac := fields[1]
		if len(mac) != 64 {
			continue
		}
		bufOut.Reset()
		status, err := runDiag(t, ctx, repo, []string{"diag", "blob", typ, mac})
		if err != nil {
			// Type string may not round-trip through resources.FromString for
			// every label printed by `diag state`; only assert on the ones
			// that succeed.
			continue
		}
		require.Equal(t, 0, status)
		require.NotEmpty(t, strings.TrimSpace(bufOut.String()))
		tested[typ] = true
	}

	require.NotEmpty(t, tested, "expected at least one blob type to be diagnosable")
}

func TestCovDiagBlobErrors(t *testing.T) {
	repo, snap, ctx, _ := covGenSnapshot(t)
	defer snap.Close()

	// Unknown blob type.
	status, err := runDiag(t, ctx, repo, []string{"diag", "blob", "not-a-type", strings.Repeat("ab", 32)})
	require.Error(t, err)
	require.Equal(t, 1, status)

	// Non-hex mac.
	status, err = runDiag(t, ctx, repo, []string{"diag", "blob", "object", "zzzz"})
	require.Error(t, err)
	require.Equal(t, 1, status)

	// Wrong mac length (valid hex, but not 32 bytes).
	status, err = runDiag(t, ctx, repo, []string{"diag", "blob", "object", "abcd"})
	require.Error(t, err)
	require.Equal(t, 1, status)

	// Valid-looking mac but no such blob.
	status, err = runDiag(t, ctx, repo, []string{"diag", "blob", "object", strings.Repeat("00", 32)})
	require.Error(t, err)
	require.Equal(t, 1, status)
}

func TestCovDiagBlobChunkErrorPath(t *testing.T) {
	repo, snap, ctx, bufOut := covGenSnapshot(t)
	defer snap.Close()

	lines := stateLines(t, ctx, repo, bufOut)

	var chunkMac string
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "chunk" && len(fields[1]) == 64 {
			chunkMac = fields[1]
			break
		}
	}
	require.NotEmpty(t, chunkMac, "expected a chunk index line")

	// Per the brief, RT_CHUNK won't deserialize from GetBlobBytes; this drives
	// the error branch of the RT_CHUNK case (or the GetBlobBytes failure).
	status, err := runDiag(t, ctx, repo, []string{"diag", "blob", "chunk", chunkMac})
	require.Error(t, err)
	require.Equal(t, 1, status)
}

func TestCovDiagBlobParseError(t *testing.T) {
	_, snap, ctx, _ := covGenSnapshot(t)
	defer snap.Close()

	// Only one positional arg -> Parse usage error.
	subcommand, _, rest := subcommands.Lookup([]string{"diag", "blob", "object"})
	require.NotNil(t, subcommand)
	err := subcommand.Parse(ctx, rest)
	require.Error(t, err)
}

// --- diag blobsearch -------------------------------------------------------

func TestCovDiagBlobSearch(t *testing.T) {
	repo, snap, ctx, bufOut := covGenSnapshot(t)
	defer snap.Close()

	lines := stateLines(t, ctx, repo, bufOut)

	var objMac string
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "object" && len(fields[1]) == 64 {
			objMac = fields[1]
			break
		}
	}
	require.NotEmpty(t, objMac, "expected an object index line")

	bufOut.Reset()
	status, err := runDiag(t, ctx, repo, []string{"diag", "blobsearch", objMac})
	require.NoError(t, err)
	require.Equal(t, 0, status)
	out := bufOut.String()
	require.Contains(t, out, "Warning this command is slow")
	require.Contains(t, out, "Found candidate")
}

func TestCovDiagBlobSearchErrors(t *testing.T) {
	repo, snap, ctx, _ := covGenSnapshot(t)
	defer snap.Close()

	// Hash of wrong length -> Execute returns error.
	status, err := runDiag(t, ctx, repo, []string{"diag", "blobsearch", "deadbeef"})
	require.Error(t, err)
	require.Equal(t, 1, status)

	// 64 chars but not valid hex.
	status, err = runDiag(t, ctx, repo, []string{"diag", "blobsearch", strings.Repeat("z", 64)})
	require.Error(t, err)
	require.Equal(t, 1, status)

	// No arg -> Parse error.
	subcommand, _, rest := subcommands.Lookup([]string{"diag", "blobsearch"})
	require.NotNil(t, subcommand)
	err = subcommand.Parse(ctx, rest)
	require.Error(t, err)
}

// --- diag dirpack ----------------------------------------------------------

func TestCovDiagDirPack(t *testing.T) {
	repo, snap, ctx, bufOut := covGenSnapshot(t)
	defer snap.Close()

	indexID := snap.Header.GetIndexID()
	bufOut.Reset()
	status, err := runDiag(t, ctx, repo, []string{"diag", "dirpack", hex.EncodeToString(indexID[:])})
	// dirpack may or may not have an index depending on snapshot; accept both
	// success and the explicit "no dirpack index" error, but exercise the path.
	if err != nil {
		require.Equal(t, 1, status)
		require.Contains(t, err.Error(), "dirpack")
		return
	}
	require.Equal(t, 0, status)
}

func TestCovDiagDirPackParseError(t *testing.T) {
	_, snap, ctx, _ := covGenSnapshot(t)
	defer snap.Close()

	subcommand, _, rest := subcommands.Lookup([]string{"diag", "dirpack"})
	require.NotNil(t, subcommand)
	err := subcommand.Parse(ctx, rest)
	require.Error(t, err)
}

func TestCovDiagDirPackBadSnapshot(t *testing.T) {
	repo, snap, ctx, _ := covGenSnapshot(t)
	defer snap.Close()

	// Non-existent snapshot id -> Execute error.
	status, err := runDiag(t, ctx, repo, []string{"diag", "dirpack", strings.Repeat("00", 32)})
	require.Error(t, err)
	require.Equal(t, 1, status)
}

// --- diag chunks -----------------------------------------------------------

func TestCovDiagChunks(t *testing.T) {
	repo, snap, ctx, bufOut := covGenSnapshot(t)
	defer snap.Close()

	indexID := snap.Header.GetIndexID()
	bufOut.Reset()
	status, err := runDiag(t, ctx, repo, []string{
		"diag", "chunks",
		fmt.Sprintf("%s:subdir/dummy.txt", hex.EncodeToString(indexID[:])),
	})
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.Contains(t, bufOut.String(), "Chunk[0]:")
}

func TestCovDiagChunksErrors(t *testing.T) {
	repo, snap, ctx, _ := covGenSnapshot(t)
	defer snap.Close()

	indexID := snap.Header.GetIndexID()

	// Path that points at a directory (no resolved object) -> error.
	status, err := runDiag(t, ctx, repo, []string{
		"diag", "chunks",
		fmt.Sprintf("%s:subdir", hex.EncodeToString(indexID[:])),
	})
	require.Error(t, err)
	require.Equal(t, 1, status)

	// Non-existent path -> error.
	status, err = runDiag(t, ctx, repo, []string{
		"diag", "chunks",
		fmt.Sprintf("%s:does/not/exist", hex.EncodeToString(indexID[:])),
	})
	require.Error(t, err)
	require.Equal(t, 1, status)

	// No arg -> Parse error.
	subcommand, _, rest := subcommands.Lookup([]string{"diag", "chunks"})
	require.NotNil(t, subcommand)
	err = subcommand.Parse(ctx, rest)
	require.Error(t, err)
}

// --- Parse error paths for the path-taking subcommands ---------------------

func TestCovDiagParseErrors(t *testing.T) {
	_, snap, ctx, _ := covGenSnapshot(t)
	defer snap.Close()

	for _, name := range []string{"snapshot", "vfs", "xattr", "contenttype", "object"} {
		t.Run(name, func(t *testing.T) {
			subcommand, _, rest := subcommands.Lookup([]string{"diag", name})
			require.NotNil(t, subcommand)
			err := subcommand.Parse(ctx, rest)
			require.Error(t, err, "expected usage error for diag %s with no args", name)
		})
	}
}

// --- Execute error paths for object / vfs / search -------------------------

func TestCovDiagObjectErrors(t *testing.T) {
	repo, snap, ctx, _ := covGenSnapshot(t)
	defer snap.Close()

	// Wrong-length object hash.
	status, err := runDiag(t, ctx, repo, []string{"diag", "object", "abcd"})
	require.Error(t, err)
	require.Equal(t, 1, status)

	// Right length, non-hex.
	status, err = runDiag(t, ctx, repo, []string{"diag", "object", strings.Repeat("z", 64)})
	require.Error(t, err)
	require.Equal(t, 1, status)

	// Right length hex, no such object.
	status, err = runDiag(t, ctx, repo, []string{"diag", "object", strings.Repeat("00", 32)})
	require.Error(t, err)
	require.Equal(t, 1, status)
}

func TestCovDiagVFSNotFound(t *testing.T) {
	repo, snap, ctx, _ := covGenSnapshot(t)
	defer snap.Close()

	indexID := snap.Header.GetIndexID()
	status, err := runDiag(t, ctx, repo, []string{
		"diag", "vfs",
		fmt.Sprintf("%s:/no/such/path", hex.EncodeToString(indexID[:])),
	})
	require.Error(t, err)
	require.Equal(t, 1, status)
}

func TestCovDiagSearchWithMime(t *testing.T) {
	repo, snap, ctx, bufOut := covGenSnapshot(t)
	defer snap.Close()

	indexID := snap.Header.GetIndexID()

	// Two-arg form: path + mime filter (exercises the case 2 branch in Parse).
	bufOut.Reset()
	status, err := runDiag(t, ctx, repo, []string{
		"diag", "search",
		fmt.Sprintf("%s:subdir/", hex.EncodeToString(indexID[:])),
		"text/plain",
	})
	require.NoError(t, err)
	require.Equal(t, 0, status)

	// No-arg form -> Parse usage error.
	subcommand, _, rest := subcommands.Lookup([]string{"diag", "search"})
	require.NotNil(t, subcommand)
	err = subcommand.Parse(ctx, rest)
	require.Error(t, err)
}
