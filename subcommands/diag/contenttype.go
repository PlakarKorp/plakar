package diag

import (
	"flag"
	"fmt"
	"strings"

	"github.com/PlakarKorp/kloset/locate"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
)

func ContentType(ctx *appcontext.AppContext, repo *repository.Repository, args []string) error {
	flags := flag.NewFlagSet("diag contenttype", flag.ExitOnError)
	flags.Parse(args)

	if len(flags.Args()) < 1 {
		return fmt.Errorf("usage: %s contenttype SNAPSHOT[:PATH]", flags.Name())
	}

	snap, pathname, err := locate.OpenSnapshotByPath(repo, flags.Args()[0])
	if err != nil {
		return err
	}
	defer snap.Close()

	if pathname == "" {
		pathname = "/"
	}
	if !strings.HasSuffix(pathname, "/") {
		pathname += "/"
	}

	tree, err := snap.ContentTypeIdx()
	if err != nil {
		return err
	}
	if tree == nil {
		return fmt.Errorf("no content-type index available in the snapshot")
	}

	it, err := tree.ScanFrom(pathname)
	if err != nil {
		return err
	}

	for it.Next() {
		path, _ := it.Current()
		if !strings.HasPrefix(path, pathname) {
			break
		}

		fmt.Fprintln(ctx.Stdout, path)
	}
	if err := it.Err(); err != nil {
		return err
	}

	return nil
}
