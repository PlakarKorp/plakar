package expose

import (
	"fmt"
	"strings"

	"github.com/PlakarKorp/kloset/connectors/visualizer"
	"github.com/PlakarKorp/kloset/locate"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/spf13/cobra"
)

type Expose struct {
	subcommands.SubcommandBase

	Location     string
	SnapshotPath string
}

func init() {
	subcommands.Register(func() subcommands.Subcommand { return &Expose{} }, 0, "expose")
}

func (cmd *Expose) CobraCommand() *cobra.Command {
	return &cobra.Command{
		Use: "expose VISUALIZER://SNAPSHOT",
	}
}

func (cmd *Expose) Parse(ctx *appcontext.AppContext, args []string) error {
	rest, err := subcommands.ParseCobra(cmd, args)
	if err != nil {
		return err
	}
	if len(rest) != 1 {
		return fmt.Errorf("expose requires one visualizer location")
	}

	protocol, snapshotPath, ok := strings.Cut(rest[0], "://")
	if !ok || protocol == "" || snapshotPath == "" {
		return fmt.Errorf("invalid visualizer location %q", rest[0])
	}

	cmd.RepositorySecret = ctx.GetSecret()
	cmd.Location = rest[0]
	cmd.SnapshotPath = snapshotPath
	return nil
}

func (cmd *Expose) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	instance, err := visualizer.NewVisualizer(ctx.GetInner(), ctx.VisualizerOpts(), map[string]string{
		"location": cmd.Location,
	})
	if err != nil {
		return 1, err
	}

	snap, _, err := locate.OpenSnapshotByPath(repo, cmd.SnapshotPath)
	if err != nil {
		return 1, err
	}
	defer snap.Close()

	pvfs, err := snap.Filesystem()
	if err != nil {
		return 1, fmt.Errorf("get snapshot filesystem: %w", err)
	}

	if err := instance.Expose(ctx.GetInner(), pvfs); err != nil {
		return 1, fmt.Errorf("expose snapshot: %w", err)
	}
	return 0, nil
}
