package diag

import (
	"fmt"

	"github.com/PlakarKorp/kloset/locate"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/spf13/cobra"
)

type DiagChunks struct {
	subcommands.SubcommandBase

	SnapshotPath string
}

func (cmd *DiagChunks) CobraCommand() *cobra.Command {
	return &cobra.Command{
		Use: "diag chunks",
	}
}

func (cmd *DiagChunks) Parse(ctx *appcontext.AppContext, args []string) error {
	rest, err := subcommands.ParseCobra(cmd, args)
	if err != nil {
		return err
	}

	if len(rest) != 1 {
		return fmt.Errorf("usage: %s chunks SNAPSHOT:PATH", "diag chunks")
	}

	cmd.RepositorySecret = ctx.GetSecret()
	cmd.SnapshotPath = rest[0]

	return nil
}

func (cmd *DiagChunks) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	snap, pathname, err := locate.OpenSnapshotByPath(repo, cmd.SnapshotPath)
	if err != nil {
		return 1, err
	}
	defer snap.Close()

	fs, err := snap.Filesystem()
	if err != nil {
		return 1, err
	}

	entry, err := fs.GetEntry(pathname)
	if err != nil {
		return 1, err
	}

	if entry.ResolvedObject == nil {
		return 1, fmt.Errorf("no object for path: %s", pathname)
	}

	var offset int64
	for i, chunk := range entry.ResolvedObject.Chunks {
		fmt.Fprintf(ctx.Stdout, "Chunk[%d]: offset=%d length=%d mac=%x entropy=%f\n",
			i, offset, chunk.Length, chunk.ContentMAC, chunk.Entropy)
		offset += int64(chunk.Length)
	}

	return 0, nil
}
