package info

import (
	"fmt"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/locate"
)

func (cmd *Info) executeErrors(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	snap, pathname, err := locate.OpenSnapshotByPath(repo, cmd.SnapshotID)
	if err != nil {
		return 1, err
	}
	defer snap.Close()

	fs, err := snap.Filesystem()
	if err != nil {
		return 1, err
	}

	errstream, err := fs.Errors(pathname)
	if err != nil {
		return 1, err
	}

	for item := range errstream {
		fmt.Fprintf(ctx.Stdout, "%s: %s\n", item.Name, item.Error)
	}
	return 0, nil
}
