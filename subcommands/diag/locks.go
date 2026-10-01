package diag

import (
	"fmt"
	"time"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/spf13/cobra"
)

type DiagLocks struct {
	subcommands.SubcommandBase
}

func (cmd *DiagLocks) CobraCommand() *cobra.Command {
	return &cobra.Command{
		Use: "diag locks",
	}
}

func (cmd *DiagLocks) Parse(ctx *appcontext.AppContext, args []string) error {
	if _, err := subcommands.ParseCobra(cmd, args); err != nil {
		return err
	}

	cmd.RepositorySecret = ctx.GetSecret()

	return nil
}

func (cmd *DiagLocks) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	locksID, err := repo.GetLocks()
	if err != nil {
		return 1, err
	}

	for _, lockID := range locksID {
		rd, err := repo.GetLock(lockID)
		if err != nil {
			fmt.Fprintf(ctx.Stderr, "Failed to fetch lock %x\n", lockID)
		}

		lock, err := repository.NewLockFromStream(rd)
		rd.Close()
		if err != nil {
			fmt.Fprintf(ctx.Stderr, "Failed to deserialize lock %x\n", lockID)
		}

		var lockType string
		if lock.Exclusive {
			lockType = "exclusive"
		} else {
			lockType = "shared"
		}

		fmt.Fprintf(ctx.Stdout, "[%x] Got %s access on %s owner %s\n", lockID, lockType, lock.Timestamp.UTC().Format(time.RFC3339), lock.Hostname)
	}

	return 0, nil
}
