package testing

import (
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/PlakarKorp/plakar/appcontext"
	cachedcmd "github.com/PlakarKorp/plakar/subcommands/cached"
	"github.com/stretchr/testify/require"
)

// StartCached runs the cached daemon in-process, listening on the cached.sock
// of the given context's CacheDir. The cached client connects to that socket
// before trying to spawn an external process, so commands relying on cached
// (sync, backup -check, ...) can be exercised from tests as long as the
// server is started before they execute.
func StartCached(t *testing.T, ctx *appcontext.AppContext) {
	srvCtx := appcontext.NewAppContextFrom(ctx)

	srv := &cachedcmd.Cached{}
	err := srv.Parse(srvCtx, []string{"-foreground", "-teardown", "5m"})
	require.NoError(t, err)

	done := make(chan struct{})
	go func() {
		defer close(done)
		srv.Execute(srvCtx, nil)
	}()

	socketPath := filepath.Join(ctx.CacheDir, "cached.sock")
	require.Eventually(t, func() bool {
		conn, err := net.Dial("unix", socketPath)
		if err != nil {
			return false
		}
		conn.Close()
		return true
	}, 10*time.Second, 5*time.Millisecond, "cached did not come up on %s", socketPath)

	t.Cleanup(func() {
		srvCtx.Close()
		<-done
		srv.Close()
	})
}
