package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/PlakarKorp/kloset/caching"
	"github.com/PlakarKorp/kloset/caching/pebble"
	"github.com/PlakarKorp/kloset/connectors/storage"
	"github.com/PlakarKorp/kloset/hashing"
	"github.com/PlakarKorp/kloset/logging"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/resources"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/kloset/versioning"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cookies"
	"github.com/PlakarKorp/plakar/login"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

func TestNewRouter(t *testing.T) {
	repo := &repository.Repository{}
	ctx := appcontext.NewAppContext()
	token := "test-token"
	mux := http.NewServeMux()
	// Make sure SetupRoutes doesn't panic, which happens when invalid routes
	// are registered
	SetupRoutes(mux, repo, ctx, token, false)
}

func TestAuthMiddleware(t *testing.T) {
	t.Run("handle error mapping", func(t *testing.T) {
		for _, c := range []struct {
			name string
			err  error
			want int
		}{
			{"not-readable -> 400", repository.ErrNotReadable, http.StatusBadRequest},
			{"blob-not-found -> 404", repository.ErrBlobNotFound, http.StatusNotFound},
			{"packfile-not-found -> 404", repository.ErrPackfileNotFound, http.StatusNotFound},
			{"fs-not-exist -> 404", fs.ErrNotExist, http.StatusNotFound},
			{"snapshot-not-found -> 404", snapshot.ErrNotFound, http.StatusNotFound},
			{"non-wrapped fs-not-exist -> 500", errors.New("boom: " + fs.ErrNotExist.Error()), http.StatusInternalServerError},
			{"unknown -> 500", errors.New("some random failure"), http.StatusInternalServerError},
			{"rate-limited -> 429", login.ErrRateLimited, http.StatusTooManyRequests},
		} {
			t.Run(c.name, func(t *testing.T) {
				req, _ := http.NewRequest("GET", "/whatever", nil)
				w := httptest.NewRecorder()
				handleError(w, req, c.err)
				require.Equal(t, c.want, w.Code)

				var body ApiErrorRes
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
				require.NotNil(t, body.Error)
			})
		}
	})
}

func Test_UnknownEndpoint(t *testing.T) {
	tmpCacheDir, err := os.MkdirTemp("", "tmp_cache")
	require.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(tmpCacheDir)
	})

	config := ptesting.NewConfiguration()

	serializedConfig, err := config.ToBytes()
	require.NoError(t, err)

	hasher := hashing.GetHasher(hashing.DEFAULT_HASHING_ALGORITHM)
	wrappedConfigRd, err := storage.Serialize(hasher, resources.RT_CONFIG, versioning.GetCurrentVersion(resources.RT_CONFIG), bytes.NewReader(serializedConfig))
	require.NoError(t, err)

	wrappedConfig, err := io.ReadAll(wrappedConfigRd)
	require.NoError(t, err)

	ctx := appcontext.NewAppContext()
	cache := caching.NewManager(pebble.Constructor(tmpCacheDir))
	defer cache.Close()
	ctx.SetCache(cache)
	ctx.CacheDir = tmpCacheDir
	ctx.SetLogger(logging.NewLogger(os.Stdout, os.Stderr))

	cookies := cookies.NewManager("/tmp/test_plakar")
	ctx.SetCookies(cookies)
	ctx.Client = "plakar-test/1.0.0"

	lstore, err := storage.Create(ctx.GetInner(), map[string]string{"location": "mock:///test/location"}, wrappedConfig)
	require.NoError(t, err)
	repo, err := repository.New(ctx.GetInner(), nil, lstore, wrappedConfig)
	if err != nil {
		t.Fatal(err)
	}
	token := ""
	mux := http.NewServeMux()
	SetupRoutes(mux, repo, ctx, token, false)

	req, err := http.NewRequest("GET", "/api/unknown_endpoint", nil)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status code 200, got %d", w.Code)
	}
}
