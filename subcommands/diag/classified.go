package diag

import (
	"fmt"
	"strings"

	"github.com/PlakarKorp/kloset/locate"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/PlakarKorp/plakar/utils"
	"github.com/spf13/cobra"
)

type DiagClassified struct {
	subcommands.SubcommandBase

	SnapshotPath string
	Analyzer     string
	Class        string
}

func (cmd *DiagClassified) CobraCommand() *cobra.Command {
	return &cobra.Command{
		Use: "diag classified",
	}
}

func (cmd *DiagClassified) Parse(ctx *appcontext.AppContext, args []string) error {
	rest, err := subcommands.ParseCobra(cmd, args)
	if err != nil {
		return err
	}

	if len(rest) != 2 {
		return fmt.Errorf("usage: %s snapshot[:path] analyzer[:class]", "diag classified")
	}

	cmd.RepositorySecret = ctx.GetSecret()
	cmd.SnapshotPath = rest[0]
	cmd.Analyzer, cmd.Class, _ = strings.Cut(rest[1], ":")
	return nil
}

func (cmd *DiagClassified) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	snap, pathname, err := locate.OpenSnapshotByPath(repo, cmd.SnapshotPath)
	if err != nil {
		return 1, err
	}
	defer snap.Close()

	for c, err := range snap.ListClassified(cmd.Analyzer, cmd.Class) {
		if err != nil {
			return 1, err
		}
		if !strings.HasPrefix(c.Path, pathname) {
			continue
		}
		fmt.Fprintf(ctx.Stdout, "%x:%s %s\n", snap.Header.Identifier[0:4], utils.SanitizeText(c.Path), utils.SanitizeText(c.Class))
	}
	return 0, nil
}
