package diag

import (
	"context"
	"fmt"
	"strings"

	"github.com/PlakarKorp/kloset/locate"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/spf13/cobra"
)

type DiagSearch struct {
	subcommands.SubcommandBase

	SnapshotPath string
	Mimes        []string
}

func (cmd *DiagSearch) CobraCommand() *cobra.Command {
	return &cobra.Command{
		Use: "diag search",
	}
}

func (cmd *DiagSearch) Parse(ctx *appcontext.AppContext, args []string) error {
	rest, err := subcommands.ParseCobra(cmd, args)
	if err != nil {
		return err
	}

	var path string
	var mimes []string

	switch len(rest) {
	case 1:
		path = rest[0]
	case 2:
		path, mimes = rest[0], strings.Split(rest[1], ",")
	default:
		return fmt.Errorf("usage: %s search snapshot[:path] mimes",
			"diag search")
	}

	cmd.RepositorySecret = ctx.GetSecret()
	cmd.SnapshotPath = path
	cmd.Mimes = mimes

	return nil
}

func (cmd *DiagSearch) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	snap, pathname, err := locate.OpenSnapshotByPath(repo, cmd.SnapshotPath)
	if err != nil {
		return 1, err
	}
	defer snap.Close()

	opts := snapshot.SearchOpts{
		Recursive: true,
		Prefix:    pathname,
		Mimes:     cmd.Mimes,
	}
	it, err := snap.Search(context.Background(), &opts)
	if err != nil {
		return 1, err
	}

	for entry, err := range it {
		if err != nil {
			return 1, err
		}
		fmt.Fprintf(ctx.Stdout, "%x:%s\n", snap.Header.Identifier[0:4], entry.Path())
	}

	return 0, nil
}
