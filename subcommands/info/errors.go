package info

import (
	"fmt"

	"github.com/PlakarKorp/kloset/locate"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
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

	// GetEntry follows symlinks, so errors beneath a linked directory are
	// listed when the snapshot is addressed through the link.
	entry, err := fs.GetEntry(pathname)
	if err != nil {
		return 1, fmt.Errorf("%s: %w", pathname, err)
	}

	for item, err := range fs.Errors(entry.Path()) {
		if err != nil {
			return 1, fmt.Errorf("failed to scan errors: %w", err)
		}

		fmt.Fprintf(ctx.Stdout, "%s: %s\n", item.Name, item.Error)
	}
	return 0, nil
}
