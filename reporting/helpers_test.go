package reporting

import (
	"io"
	"net/http"
	"testing"

	"github.com/PlakarKorp/kloset/logging"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cookies"
)

// newReporterCtx returns an AppContext wired with a logger and a cookies manager
// backed by a temp dir (so getEmitter can query the auth token without panicking;
// with no token configured it returns the empty string).
func newReporterCtx(t *testing.T) *appcontext.AppContext {
	t.Helper()
	ctx := appcontext.NewAppContext()
	ctx.SetLogger(logging.NewLogger(io.Discard, io.Discard))
	ctx.SetCookies(cookies.NewManager(t.TempDir()))
	// Ensure no env token leaks in from the host.
	t.Setenv("PLAKAR_TOKEN", "")
	return ctx
}

func readAll(r *http.Request) ([]byte, error) {
	return io.ReadAll(r.Body)
}
