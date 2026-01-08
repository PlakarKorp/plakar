package api

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/PlakarKorp/kloset/caching"
	"github.com/PlakarKorp/kloset/caching/pebble"
	"github.com/PlakarKorp/kloset/hashing"
	"github.com/PlakarKorp/kloset/logging"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/resources"
	"github.com/PlakarKorp/kloset/storage"
	"github.com/PlakarKorp/kloset/versioning"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cookies"
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
	SetupRoutes(mux, repo, ctx, token)
}

func TestAuthMiddleware(t *testing.T) {
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
	token := "test-token"
	mux := http.NewServeMux()
	SetupRoutes(mux, repo, ctx, token)

	req, err := http.NewRequest("GET", "/api/info", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Invalid Token")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status code 401, got %d", w.Code)
	}

	req.Header.Set("Authorization", "")
	w2 := httptest.NewRecorder()
	mux.ServeHTTP(w2, req)
	if w2.Code != http.StatusUnauthorized {
		t.Errorf("Expected status code 401, got %d", w2.Code)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	w3 := httptest.NewRecorder()
	mux.ServeHTTP(w3, req)

	if w3.Code != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", w3.Code)
	}
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
	SetupRoutes(mux, repo, ctx, token)

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
