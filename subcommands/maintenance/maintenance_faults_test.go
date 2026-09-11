package maintenance

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/PlakarKorp/kloset/caching"
	"github.com/PlakarKorp/kloset/caching/pebble"
	"github.com/PlakarKorp/kloset/connectors/storage"
	"github.com/PlakarKorp/kloset/hashing"
	"github.com/PlakarKorp/kloset/logging"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/resources"
	"github.com/PlakarKorp/kloset/versioning"
	"github.com/PlakarKorp/plakar/appcontext"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

// faultRepo builds a repository over the mock storage backend with the given
// fault-injection behavior. repository.New only fails when List(State) errors
// (behavior=brokenState), so any behavior whose List(State) succeeds yields a
// usable repository whose later storage operations fail on demand.
func faultRepo(t *testing.T, behavior string) (*repository.Repository, *appcontext.AppContext) {
	t.Helper()

	loc := "mock:///test/loc"
	if behavior != "" {
		loc += "?behavior=" + behavior
	}

	ctx := appcontext.NewAppContext()
	t.Cleanup(ctx.Close)
	ctx.MaxConcurrency = 1

	tmpCacheDir, err := os.MkdirTemp("", "tmp_cache_fault")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(tmpCacheDir) })
	ctx.CacheDir = tmpCacheDir

	cache := caching.NewManager(pebble.Constructor(tmpCacheDir))
	t.Cleanup(func() { cache.Close() })
	ctx.SetCache(cache)
	ctx.SetLogger(logging.NewLogger(io.Discard, io.Discard))

	config := ptesting.NewConfiguration()
	serializedConfig, err := config.ToBytes()
	require.NoError(t, err)

	hasher := hashing.GetHasher(hashing.DEFAULT_HASHING_ALGORITHM)
	wrappedConfigRd, err := storage.Serialize(hasher, resources.RT_CONFIG, versioning.GetCurrentVersion(resources.RT_CONFIG), bytes.NewReader(serializedConfig))
	require.NoError(t, err)
	wrappedConfig, err := io.ReadAll(wrappedConfigRd)
	require.NoError(t, err)

	lstore, err := storage.Create(ctx.GetInner(), map[string]string{"location": loc}, wrappedConfig)
	require.NoError(t, err, "creating storage")

	repo, err := repository.New(ctx.GetInner(), nil, lstore, wrappedConfig)
	require.NoError(t, err, "creating repository")
	return repo, ctx
}

// TestFaultColourPassGetPackfilesError drives colourPass against a backend whose
// List(Packfile) (repository.GetPackfiles) returns an error, exercising the
// `repoPackfiles, err := cmd.repository.GetPackfiles()` error return.
func TestFaultColourPassGetPackfilesError(t *testing.T) {
	resetEnv(t)
	repo, ctx := faultRepo(t, "brokenGetPackfiles")

	cmd := &Maintenance{}
	require.NoError(t, cmd.Parse(ctx, nil))
	cmd.repository = repo

	cache, err := repo.AppContext().GetCache().Maintenance(repo.Configuration().RepositoryID)
	require.NoError(t, err)

	err = cmd.colourPass(ctx, cache)
	require.Error(t, err)
	require.Contains(t, err.Error(), "broken get packfiles")
}

// TestFaultColourPassGetPackfileError drives colourPass against a backend that
// lists orphan packfiles (present in storage but referenced by no state) and
// errors on Get(Packfile). The orphan branch loads the packfile body,
// exercising the `packfile, err := cmd.repository.GetPackfile(...)` error
// return.
func TestFaultColourPassGetPackfileError(t *testing.T) {
	resetEnv(t)
	repo, ctx := faultRepo(t, "orphanBrokenGetPackfile")

	cmd := &Maintenance{}
	require.NoError(t, cmd.Parse(ctx, nil))
	cmd.repository = repo

	cache, err := repo.AppContext().GetCache().Maintenance(repo.Configuration().RepositoryID)
	require.NoError(t, err)

	err = cmd.colourPass(ctx, cache)
	require.Error(t, err)
	require.Contains(t, err.Error(), "broken get packfile")
}

// TestFaultExecuteColourFailsReturnsError runs the full Execute path over a
// backend whose List(Packfile) errors, exercising the
// `if err := cmd.colourPass(...); err != nil { return 1, err }` branch.
func TestFaultExecuteColourFailsReturnsError(t *testing.T) {
	resetEnv(t)
	t.Setenv("PLAKAR_LOCKLESS", "true")
	repo, ctx := faultRepo(t, "brokenGetPackfiles")

	cmd := &Maintenance{}
	require.NoError(t, cmd.Parse(ctx, nil))

	status, err := cmd.Execute(ctx, repo)
	require.Error(t, err)
	require.Equal(t, 1, status)
	require.Contains(t, err.Error(), "broken get packfiles")
}
