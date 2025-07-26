package diag

import (
	"flag"
	"fmt"
	"strings"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/locate"
	"github.com/PlakarKorp/plakar/subcommands"
)

type DiagContentType struct {
	subcommands.SubcommandBase

	SnapshotPath string
}

func (cmd *DiagContentType) Parse(ctx *appcontext.AppContext, args []string) error {
	flags := flag.NewFlagSet("diag contenttype", flag.ExitOnError)
	flags.Parse(args)

	if len(flags.Args()) < 1 {
		return fmt.Errorf("usage: %s contenttype SNAPSHOT[:PATH]", flags.Name())
	}

	cmd.RepositorySecret = ctx.GetSecret()
	cmd.SnapshotPath = flags.Args()[0]

	return nil
}

func (cmd *DiagContentType) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	snap, pathname, err := locate.OpenSnapshotByPath(repo, cmd.SnapshotPath)
	if err != nil {
		return 1, err
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
		return 1, err
	}
	if tree == nil {
		return 1, fmt.Errorf("no content-type index available in the snapshot")
	}

	it, err := tree.ScanFrom(pathname)
	if err != nil {
		return 1, err
	}

	for it.Next() {
		path, _ := it.Current()
		if !strings.HasPrefix(path, pathname) {
			break
		}

		fmt.Fprintln(ctx.Stdout, path)
	}
	if err := it.Err(); err != nil {
		return 1, err
	}

	return 0, nil
}
