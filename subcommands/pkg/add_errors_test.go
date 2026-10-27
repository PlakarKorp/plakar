package pkg

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/PlakarKorp/kloset/logging"
	ppkg "github.com/PlakarKorp/pkg"
	"github.com/PlakarKorp/plakar/signify"
	"github.com/stretchr/testify/require"
)

// The user sees which package failed and why; the repository URL only goes
// to the trace.
func TestPkgAddErrorMessages(t *testing.T) {
	dir := path.Join("/community/", ppkg.PLUGIN_API_VERSION, "s3")
	recipe := path.Join(dir, "recipe.yaml")
	const recipeBody = "name: s3\nversion: v2.0.0\nrepository: https://example.com/s3\n"

	serve := func(status int) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != recipe {
				http.NotFound(w, r)
				return
			}
			if status != http.StatusOK {
				w.WriteHeader(status)
				return
			}
			fmt.Fprint(w, recipeBody)
		}
	}
	verifyWith := func(cause error) ppkg.Verifier {
		return ppkg.VerifierFunc(func(a *ppkg.Artifact, rd io.Reader) error {
			if cause != nil {
				return fmt.Errorf("%w: %w", ppkg.ErrUnverified, cause)
			}
			_, err := io.Copy(io.Discard, rd)
			return err
		})
	}

	tests := []struct {
		name     string
		handler  http.HandlerFunc
		verifier ppkg.Verifier
		down     bool
		want     string
	}{
		{"no such package", serve(http.StatusNotFound), nil, false,
			"failed to install s3: no such package in the community edition"},
		{"no build for platform", serve(http.StatusOK), nil, false,
			fmt.Sprintf("failed to install s3@v2.0.0: not available for %s/%s", runtime.GOOS, runtime.GOARCH)},
		{"unauthorized", serve(http.StatusUnauthorized), nil, false,
			"failed to install s3: authentication required. Run `plakar login` and try again"},
		{"forbidden", serve(http.StatusForbidden), nil, false,
			"failed to install s3: access denied by the package repository"},
		{"server error", serve(http.StatusInternalServerError), nil, false,
			"failed to install s3: the package repository answered 500 Internal Server Error while fetching the recipe"},
		{"unsigned", serve(http.StatusOK), verifyWith(signify.ErrUnsigned), false,
			"failed to install s3: the recipe is not signed"},
		{"bad signature", serve(http.StatusOK), verifyWith(signify.ErrDigestMismatch), false,
			"failed to install s3: the recipe does not match its signature"},
		{"unreachable", serve(http.StatusOK), nil, true,
			"failed to install s3: could not fetch the recipe"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(tt.handler)
			defer srv.Close()
			if tt.down {
				srv.Close()
			}

			ctx := newCtx(t)
			var trace bytes.Buffer
			logger := logging.NewLogger(&trace, &trace)
			logger.EnableTracing("pkg")
			ctx.SetLogger(logger)

			backend, err := ppkg.NewFlatBackend(ctx.GetInner(),
				filepath.Join(ctx.CWD, "plugins"), filepath.Join(ctx.CWD, "cache"),
				&ppkg.FlatBackendOptions{})
			require.NoError(t, err)

			manager, err := ppkg.New(backend, &ppkg.Options{DistURL: srv.URL, Verifier: tt.verifier})
			require.NoError(t, err)
			ctx.SetPkgManager(manager)

			cmd := &PkgAdd{}
			require.NoError(t, cmd.Parse(ctx, []string{"s3"}))

			status, err := cmd.Execute(ctx, nil)
			require.Equal(t, 1, status)
			require.EqualError(t, err, tt.want)
			require.NotContains(t, err.Error(), srv.URL)
			require.Contains(t, trace.String(), srv.URL)
		})
	}
}
