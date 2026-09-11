package help

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sync"
	"testing"

	bfs "github.com/PlakarKorp/integrations/fs/storage"
	"github.com/PlakarKorp/kloset/caching"
	"github.com/PlakarKorp/kloset/caching/pebble"
	"github.com/PlakarKorp/kloset/connectors/storage"
	"github.com/PlakarKorp/kloset/hashing"
	"github.com/PlakarKorp/kloset/logging"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/resources"
	"github.com/PlakarKorp/kloset/versioning"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/stretchr/testify/require"
)

func TestHelpExecuteUnknownCommand(t *testing.T) {
	// An unknown command has no embedded doc, so ReadFile fails and Execute
	// returns 1.
	ctx := appcontext.NewAppContext()
	cmd := &Help{}
	require.NoError(t, cmd.Parse(ctx, []string{"no-such-command"}))
	status, err := cmd.Execute(ctx, &repository.Repository{})
	require.Error(t, err)
	require.Equal(t, 1, status)
}

func TestHelpExecuteNoColor(t *testing.T) {
	// NO_COLOR forces the ascii (disableColors) rendering branch.
	t.Setenv("NO_COLOR", "1")

	old := os.Stdout
	_, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w
	defer func() { os.Stdout = old; w.Close() }()

	ctx := appcontext.NewAppContext()
	cmd := &Help{}
	require.NoError(t, cmd.Parse(ctx, []string{}))
	status, err := cmd.Execute(ctx, &repository.Repository{})
	require.NoError(t, err)
	require.Equal(t, 0, status)
}

func TestParseCmdHelpCommand(t *testing.T) {
	// Create a pipe to capture stdout
	old := os.Stdout
	r1, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	// Restore stdout after the test
	defer func() {
		os.Stdout = old
		w.Close()
	}()

	// Use a WaitGroup to wait for the read to complete
	var wg sync.WaitGroup
	wg.Add(1)

	// Capture the output in a buffer
	var buf bytes.Buffer
	go func() {
		defer wg.Done()
		io.Copy(&buf, r1)
	}()

	tmpRepoDirRoot, err := os.MkdirTemp("", "tmp_repo")
	require.NoError(t, err)
	tmpRepoDir := fmt.Sprintf("%s/repo", tmpRepoDirRoot)
	tmpCacheDir, err := os.MkdirTemp("", "tmp_cache")
	require.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(tmpCacheDir)
		os.RemoveAll(tmpRepoDir)
		os.RemoveAll(tmpRepoDirRoot)
	})

	ctx := appcontext.NewAppContext()

	// create a storage
	r, err := bfs.NewStore(ctx, "fs", map[string]string{"location": tmpRepoDir})
	require.NotNil(t, r)
	require.NoError(t, err)
	config := storage.NewConfiguration()
	serialized, err := config.ToBytes()
	require.NoError(t, err)

	hasher := hashing.GetHasher(hashing.DEFAULT_HASHING_ALGORITHM)
	wrappedConfigRd, err := storage.Serialize(hasher, resources.RT_CONFIG, versioning.GetCurrentVersion(resources.RT_CONFIG), bytes.NewReader(serialized))
	require.NoError(t, err)

	wrappedConfig, err := io.ReadAll(wrappedConfigRd)
	require.NoError(t, err)

	err = r.Create(ctx, wrappedConfig)
	require.NoError(t, err)

	// open the storage to load the configuration
	r, serializedConfig, err := storage.Open(ctx.GetInner(), map[string]string{"location": tmpRepoDir})
	require.NoError(t, err)

	// create a repository
	cache := caching.NewManager(pebble.Constructor(tmpCacheDir))
	ctx.SetCache(cache)
	ctx.CacheDir = tmpCacheDir
	ctx.Client = "plakar-test/1.0.0"

	// Create a new logger
	logger := logging.NewLogger(bytes.NewBuffer(nil), bytes.NewBuffer(nil))
	logger.EnableInfo()
	ctx.SetLogger(logger)
	repo, err := repository.New(ctx.GetInner(), nil, r, serializedConfig)
	require.NoError(t, err)
	args := []string{"-style", "notty", "version"}

	subcommand := &Help{}
	err = subcommand.Parse(ctx, args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	status, err := subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	// Close the write end of the pipe to signal EOF to the reader
	w.Close()

	// Wait for the read to complete
	wg.Wait()

	// Restore stdout
	os.Stdout = old

	output := buf.String()
	require.Contains(t, output, "PLAKAR-VERSION(1) - General Commands Manual")
}
