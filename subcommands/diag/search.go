package diag

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/PlakarKorp/kloset/locate"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/plakar/appcontext"
)

func Search(ctx *appcontext.AppContext, repo *repository.Repository, args []string) error {
	flags := flag.NewFlagSet("diag search", flag.ExitOnError)
	flags.Parse(args)

	var (
		path  string
		mimes []string
	)

	switch flags.NArg() {
	case 1:
		path = flags.Arg(0)
	case 2:
		path, mimes = flags.Arg(0), strings.Split(flags.Arg(1), ",")
	default:
		return fmt.Errorf("usage: %s search snapshot[:path] mimes",
			flags.Name())
	}

	snap, pathname, err := locate.OpenSnapshotByPath(repo, path)
	if err != nil {
		return err
	}
	defer snap.Close()

	opts := snapshot.SearchOpts{
		Recursive: true,
		Prefix:    pathname,
		Mimes:     mimes,
	}
	it, err := snap.Search(context.Background(), &opts)
	if err != nil {
		return err
	}

	for entry, err := range it {
		if err != nil {
			return err
		}
		fmt.Fprintf(ctx.Stdout, "%x:%s\n", snap.Header.Identifier[0:4], entry.Path())
	}

	return nil
}
