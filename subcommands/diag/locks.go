package diag

import (
	"flag"
	"fmt"
	"time"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
)

func Locks(ctx *appcontext.AppContext, repo *repository.Repository, args []string) error {
	flags := flag.NewFlagSet("diag locks", flag.ExitOnError)
	flags.Parse(args)

	locksID, err := repo.GetLocks()
	if err != nil {
		return err
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

	return nil
}
