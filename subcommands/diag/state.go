package diag

import (
	"encoding/hex"
	"flag"
	"fmt"

	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/repository/state"
	"github.com/PlakarKorp/kloset/resources"
	"github.com/PlakarKorp/plakar/appcontext"
)

func State(ctx *appcontext.AppContext, repo *repository.Repository, args []string) error {
	flags := flag.NewFlagSet("diag state", flag.ExitOnError)
	flags.Parse(args)

	if len(flags.Args()) == 0 {
		states, err := repo.GetStates()
		if err != nil {
			return err
		}

		for _, state := range states {
			fmt.Fprintf(ctx.Stdout, "%x\n", state)
		}
	} else {
		for _, arg := range flags.Args() {
			// convert arg to [32]byte
			if len(arg) != 64 {
				return fmt.Errorf("invalid packfile hash: %s", arg)
			}

			b, err := hex.DecodeString(arg)
			if err != nil {
				return fmt.Errorf("invalid packfile hash: %s", arg)
			}

			// Convert the byte slice to a [32]byte
			var byteArray [32]byte
			copy(byteArray[:], b)

			rawStateRd, v, err := repo.GetState(byteArray)
			if err != nil {
				return err
			}
			defer rawStateRd.Close()

			// Temporary scan cache to reconstruct that state.
			identifier := objects.RandomMAC()
			scanCache, err := repo.AppContext().GetCache().Scan(identifier)
			if err != nil {
				return err
			}
			defer scanCache.Close()

			st, err := state.FromStream(rawStateRd, v, scanCache)
			if err != nil {
				return err
			}

			fmt.Fprintf(ctx.Stdout, "Version: %s\n", st.Metadata.Version)
			fmt.Fprintf(ctx.Stdout, "Creation: %s\n", st.Metadata.Timestamp)
			fmt.Fprintf(ctx.Stdout, "State serial: %s\n", st.Metadata.Serial)

			printBlobs := func(name string, Type resources.Type) {
				for snapshot, err := range st.ListObjectsOfType(Type) {
					if err != nil {
						fmt.Fprintf(ctx.Stdout, "Could not fetch blob entry for %s\n", name)
					} else {
						fmt.Fprintf(ctx.Stdout, "%s %x : packfile %x, offset %d, length %d\n",
							name,
							snapshot.Blob,
							snapshot.Location.Packfile,
							snapshot.Location.Offset,
							snapshot.Location.Length)
					}
				}
			}
			printDeleted := func(name string, Type resources.Type) {
				for deletedEntry, err := range st.ListColouredResources(Type) {
					if err != nil {
						fmt.Fprintf(ctx.Stdout, "Could not fetch deleted blob entry for %s\n", name)
					} else {
						fmt.Fprintf(ctx.Stdout, "deleted %s: %x, when=%s\n",
							name,
							deletedEntry.Blob,
							deletedEntry.When)
					}
				}
			}

			for _, Type := range resources.Types() {
				printDeleted(Type.String(), Type)
				printBlobs(Type.String(), Type)
			}
		}
	}
	return nil
}
