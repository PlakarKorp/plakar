package cached

import (
	"time"

	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/google/uuid"
)

func RebuildStateFromStateFile(ctx *appcontext.AppContext, stateID objects.MAC, repoID uuid.UUID, storeConfig map[string]string, fireAndForget bool) (int, error) {
	t0 := time.Now()
	defer func() {
		ctx.GetLogger().Trace("cached", "rebuild from local statefile (file=%x, store=%s): %s", stateID, repoID, time.Since(t0))
	}()

	req := &RequestPkt{
		Secret:        ctx.GetSecret(),
		RepoID:        repoID,
		StoreConfig:   storeConfig,
		StateID:       stateID,
		FireAndForget: fireAndForget,
	}

	return rebuildStateRequest(ctx, req)
}

func RebuildStateFromStore(ctx *appcontext.AppContext, repoID uuid.UUID, storeConfig map[string]string, fireAndForget bool) (int, error) {
	t0 := time.Now()
	defer func() {
		ctx.GetLogger().Trace("cached", "rebuild from store (store=%s): %s", repoID, time.Since(t0))
	}()
	req := &RequestPkt{
		Secret:        ctx.GetSecret(),
		RepoID:        repoID,
		StoreConfig:   storeConfig,
		FireAndForget: fireAndForget,
	}

	return rebuildStateRequest(ctx, req)
}
