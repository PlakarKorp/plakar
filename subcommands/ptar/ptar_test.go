package ptar

import (
	"bytes"
	"context"
	"encoding/hex"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"

	bfs "github.com/PlakarKorp/integrations/fs/storage"
	_ "github.com/PlakarKorp/integrations/ptar/storage"
	"github.com/PlakarKorp/kloset/connectors/storage"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/config"
	lscmd "github.com/PlakarKorp/plakar/subcommands/ls"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

var capturedPtarSyncStoreConfigs []map[string]string

func init() {
	os.Setenv("TZ", "UTC")
	// several fixtures below use a plaintext -k source on purpose
	os.Setenv("PLAKAR_INSECURE_PLAINTEXT", "1")

	storage.Register("ptartestfs", 0, func(ctx context.Context, proto string, storeConfig map[string]string) (storage.Store, error) {
		capturedPtarSyncStoreConfigs = append(capturedPtarSyncStoreConfigs, maps.Clone(storeConfig))

		fsConfig := maps.Clone(storeConfig)
		fsConfig["location"] = "fs://" + storeConfig["location"][len("ptartestfs://"):]
		return bfs.NewStore(ctx, "fs", fsConfig)
	})
}

func TestExecuteCmdPtarDefault(t *testing.T) {
	repo, ctx := ptesting.GenerateRepositoryWithoutConfig(t, nil, nil, nil)
	tmpSourceDir := ptesting.GenerateFiles(t, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockDir("another_subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
		ptesting.NewMockFile("subdir/foo.txt", 0644, "hello foo"),
		ptesting.NewMockFile("subdir/to_exclude", 0644, "*/subdir/to_exclude\n"),
		ptesting.NewMockFile("another_subdir/bar.txt", 0644, "hello bar"),
	})

	tmpDir, err := os.MkdirTemp("", "tmp_ptar")
	require.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})

	args := []string{"-plaintext", "-o", filepath.Join(tmpDir, "test.ptar"), filepath.Join(tmpSourceDir, "subdir")}

	subcommand := &Ptar{}
	err = subcommand.Parse(ctx, args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	status, err := subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
}

func TestExecuteCmdPtarWithIgnoreRules(t *testing.T) {
	repo, ctx := ptesting.GenerateRepositoryWithoutConfig(t, nil, nil, nil)
	tmpSourceDir := ptesting.GenerateFiles(t, []ptesting.MockFile{
		ptesting.NewMockDir("project"),
		ptesting.NewMockDir("project/.git"),
		ptesting.NewMockDir("project/.venv"),
		ptesting.NewMockFile("project/readme.md", 0644, "hello"),
		ptesting.NewMockFile("project/.git/config", 0644, "hidden"),
		ptesting.NewMockFile("project/.venv/site.py", 0644, "hidden"),
	})
	tmpDir := t.TempDir()
	out := filepath.Join(tmpDir, "ignored.ptar")
	ignoreFile := filepath.Join(tmpDir, "ptar-ignore")
	require.NoError(t, os.WriteFile(ignoreFile, []byte(".venv\n"), 0600))

	cmd := &Ptar{}
	require.NoError(t, cmd.Parse(ctx, []string{
		"-plaintext",
		"-ignore", ".git",
		"-ignore-file", ignoreFile,
		"-o", out,
		filepath.Join(tmpSourceDir, "project"),
	}))

	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	output := listPtarContents(t, ctx, out)
	require.Contains(t, output, "/project/readme.md")
	require.NotContains(t, output, ".git")
	require.NotContains(t, output, ".venv")
}

func TestExecuteCmdPtarWithSync(t *testing.T) {
	// Create source repository
	srcRepo, _ := ptesting.GenerateRepository(t, nil, nil, nil)
	srcSnap := ptesting.GenerateSnapshot(t, srcRepo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockDir("another_subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
		ptesting.NewMockFile("subdir/foo.txt", 0644, "hello foo"),
		ptesting.NewMockFile("subdir/to_exclude", 0644, "*/subdir/to_exclude\n"),
		ptesting.NewMockFile("another_subdir/bar.txt", 0644, "hello bar"),
	})
	defer srcSnap.Close()

	// Create destination repository
	dstRepo, ctx := ptesting.GenerateRepositoryWithoutConfig(t, nil, nil, nil)

	tmpDir, err := os.MkdirTemp("", "tmp_ptar")
	require.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})

	args := []string{"-plaintext", "-o", filepath.Join(tmpDir, "test.ptar"), "-k", srcRepo.Root()}

	subcommand := &Ptar{}
	err = subcommand.Parse(ctx, args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	status, err := subcommand.Execute(ctx, dstRepo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
}

func TestPtarSyncPassphraseNeverReachesTheBackend(t *testing.T) {
	capturedPtarSyncStoreConfigs = nil

	passphrase := []byte("QsDfG654321&^%*%!")
	srcRepo, _ := ptesting.GenerateRepository(t, nil, nil, &passphrase)
	snap := ptesting.GenerateSnapshot(t, srcRepo, []ptesting.MockFile{
		ptesting.NewMockFile("file.txt", 0644, "hello"),
	})
	defer snap.Close()

	dstRepo, ctx := ptesting.GenerateRepositoryWithoutConfig(t, nil, nil, nil)
	ctx.Config = config.NewConfig()
	ctx.Config.Repositories["peer"] = map[string]string{
		"location": "ptartestfs://" + srcRepo.Root(),
	}
	t.Setenv("PLAKAR_REPOSITORIES_PEER_PASSPHRASE", string(passphrase))

	cmd := &Ptar{}
	require.NoError(t, cmd.Parse(ctx, []string{
		"-plaintext", "-o", filepath.Join(t.TempDir(), "test.ptar"), "-k", "@peer",
	}))

	status, err := cmd.Execute(ctx, dstRepo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	require.NotEmpty(t, capturedPtarSyncStoreConfigs, "the sync source store was never opened")
	for i, cfg := range capturedPtarSyncStoreConfigs {
		_, hasPass := cfg["passphrase"]
		_, hasPassCmd := cfg["passphrase_cmd"]
		require.False(t, hasPass, "call #%d: backend received the peer passphrase in cleartext", i)
		require.False(t, hasPassCmd, "call #%d: backend received passphrase_cmd", i)
	}
}

func TestExecuteCmdPtarSyncFiltersSnapshots(t *testing.T) {
	// The locate flags select which snapshots of a -k kloset are pulled in.
	srcRepo, _ := ptesting.GenerateRepository(t, nil, nil, nil)
	kept := ptesting.GenerateSnapshot(t, srcRepo, []ptesting.MockFile{
		ptesting.NewMockDir("kept"),
		ptesting.NewMockFile("kept/file.txt", 0644, "keep me"),
	}, ptesting.WithName("keep-me"))
	defer kept.Close()
	dropped := ptesting.GenerateSnapshot(t, srcRepo, []ptesting.MockFile{
		ptesting.NewMockDir("dropped"),
		ptesting.NewMockFile("dropped/file.txt", 0644, "drop me"),
	}, ptesting.WithName("drop-me"))
	defer dropped.Close()

	dstRepo, ctx := ptesting.GenerateRepositoryWithoutConfig(t, nil, nil, nil)
	out := filepath.Join(t.TempDir(), "filtered.ptar")

	cmd := &Ptar{}
	require.NoError(t, cmd.Parse(ctx, []string{
		"-plaintext", "-o", out, "-k", srcRepo.Root(), "-name", "keep-me",
	}))

	status, err := cmd.Execute(ctx, dstRepo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	ids := listPtarSnapshotIDs(t, ctx, out)
	require.Equal(t, []string{hex.EncodeToString(kept.Header.GetIndexShortID())}, ids)
}

func TestExecuteCmdPtarSyncKeepsEverySnapshotByDefault(t *testing.T) {
	// Without a filter the whole kloset is still pulled in.
	srcRepo, _ := ptesting.GenerateRepository(t, nil, nil, nil)
	first := ptesting.GenerateSnapshot(t, srcRepo, []ptesting.MockFile{
		ptesting.NewMockDir("one"),
		ptesting.NewMockFile("one/file.txt", 0644, "first"),
	}, ptesting.WithName("first"))
	defer first.Close()
	second := ptesting.GenerateSnapshot(t, srcRepo, []ptesting.MockFile{
		ptesting.NewMockDir("two"),
		ptesting.NewMockFile("two/file.txt", 0644, "second"),
	}, ptesting.WithName("second"))
	defer second.Close()

	dstRepo, ctx := ptesting.GenerateRepositoryWithoutConfig(t, nil, nil, nil)
	out := filepath.Join(t.TempDir(), "everything.ptar")

	cmd := &Ptar{}
	require.NoError(t, cmd.Parse(ctx, []string{
		"-plaintext", "-o", out, "-k", srcRepo.Root(),
	}))

	status, err := cmd.Execute(ctx, dstRepo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	ids := listPtarSnapshotIDs(t, ctx, out)
	require.ElementsMatch(t, []string{
		hex.EncodeToString(first.Header.GetIndexShortID()),
		hex.EncodeToString(second.Header.GetIndexShortID()),
	}, ids)
}

// listPtarSnapshotIDs returns the short id of every snapshot held in a ptar.
func listPtarSnapshotIDs(t *testing.T, ctx *appcontext.AppContext, path string) []string {
	t.Helper()

	storeConfig := map[string]string{"location": "ptar://" + path}
	st, serializedConfig, err := storage.Open(ctx.GetInner(), storeConfig)
	require.NoError(t, err)
	defer st.Close(ctx.GetInner())

	repo, err := repository.New(ctx.GetInner(), nil, st, serializedConfig)
	require.NoError(t, err)

	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	ctx.Stdout = stdout
	ctx.Stderr = stderr
	cmd := &lscmd.Ls{}
	require.NoError(t, cmd.Parse(ctx, nil))
	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.Empty(t, strings.TrimSpace(stderr.String()))

	ids := []string{}
	for _, line := range strings.Split(strings.TrimSpace(stdout.String()), "\n") {
		if fields := strings.Fields(line); len(fields) >= 2 {
			ids = append(ids, fields[1])
		}
	}
	return ids
}

func listPtarContents(t *testing.T, ctx *appcontext.AppContext, path string) string {
	t.Helper()

	storeConfig := map[string]string{"location": "ptar://" + path}
	st, serializedConfig, err := storage.Open(ctx.GetInner(), storeConfig)
	require.NoError(t, err)
	defer st.Close(ctx.GetInner())

	repo, err := repository.New(ctx.GetInner(), nil, st, serializedConfig)
	require.NoError(t, err)

	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	ctx.Stdout = stdout
	ctx.Stderr = stderr
	cmd := &lscmd.Ls{}
	require.NoError(t, cmd.Parse(ctx, nil))
	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.Empty(t, strings.TrimSpace(stderr.String()))

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	require.NotEmpty(t, lines)
	fields := strings.Fields(lines[0])
	require.GreaterOrEqual(t, len(fields), 2)
	snapshotID := fields[1]

	stdout.Reset()
	stderr.Reset()
	cmd = &lscmd.Ls{}
	require.NoError(t, cmd.Parse(ctx, []string{"-recursive", snapshotID}))
	status, err = cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.Empty(t, strings.TrimSpace(stderr.String()))

	return stdout.String()
}
