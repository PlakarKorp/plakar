package diag

import (
	"fmt"
	"strings"

	"github.com/PlakarKorp/kloset/locate"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/spf13/cobra"
)

type DiagContentType struct {
	subcommands.SubcommandBase

	SnapshotPath string
}

func (cmd *DiagContentType) CobraCommand() *cobra.Command {
	return &cobra.Command{
		Use: "diag contenttype",
	}
}

func (cmd *DiagContentType) Parse(ctx *appcontext.AppContext, args []string) error {
	rest, err := subcommands.ParseCobra(cmd, args)
	if err != nil {
		return err
	}

	if len(rest) < 1 {
		return fmt.Errorf("usage: %s contenttype SNAPSHOT[:PATH]", "diag contenttype")
	}

	cmd.RepositorySecret = ctx.GetSecret()
	cmd.SnapshotPath = rest[0]

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
