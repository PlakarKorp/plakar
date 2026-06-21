package info

import (
	"fmt"

	"github.com/PlakarKorp/kloset/locate"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
)

func infoErrors(ctx *appcontext.AppContext, repo *repository.Repository, snapshotID string) error {
	snap, pathname, err := locate.OpenSnapshotByPath(repo, snapshotID)
	if err != nil {
		return err
	}
	defer snap.Close()

	fs, err := snap.Filesystem()
	if err != nil {
		return err
	}

	for item, err := range fs.Errors(pathname) {
		if err != nil {
			return fmt.Errorf("failed to scan errors: %w", err)
		}

		fmt.Fprintf(ctx.Stdout, "%s: %s\n", item.Name, item.Error)
	}
	return nil
}
