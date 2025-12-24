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
	"github.com/PlakarKorp/plakar/subcommands"
)

type DiagState struct {
	subcommands.SubcommandBase

	Args []string
}

func (cmd *DiagState) Parse(ctx *appcontext.AppContext, args []string) error {
	flags := flag.NewFlagSet("diag state", flag.ExitOnError)
	flags.Parse(args)

	cmd.RepositorySecret = ctx.GetSecret()
	cmd.Args = flags.Args()

	return nil
}

func (cmd *DiagState) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	if len(cmd.Args) == 0 {
		states, err := repo.GetStates()
		if err != nil {
			return 1, err
		}

		for _, state := range states {
			fmt.Printf("%x\n", state)
		}
	} else {
		for _, arg := range cmd.Args {
			// convert arg to [32]byte
			if len(arg) != 64 {
				return 1, fmt.Errorf("invalid packfile hash: %s", arg)
			}

			b, err := hex.DecodeString(arg)
			if err != nil {
				return 1, fmt.Errorf("invalid packfile hash: %s", arg)
			}

			// Convert the byte slice to a [32]byte
			var byteArray [32]byte
			copy(byteArray[:], b)

			rawStateRd, err := repo.GetState(byteArray)
			if err != nil {
				return 1, err
			}
			defer rawStateRd.Close()

			// Temporary scan cache to reconstruct that state.
			identifier := objects.RandomMAC()
			scanCache, err := repo.AppContext().GetCache().Scan(identifier)
			if err != nil {
				return 1, err
			}
			defer scanCache.Close()

			st, err := state.FromStream(rawStateRd, scanCache)
			if err != nil {
				return 1, err
			}

			fmt.Printf("Version: %s\n", st.Metadata.Version)
			fmt.Printf("Creation: %s\n", st.Metadata.Timestamp)
			fmt.Printf("State serial: %s\n", st.Metadata.Serial)

			printBlobs := func(name string, Type resources.Type) {
				for snapshot, err := range st.ListObjectsOfType(Type) {
					if err != nil {
						fmt.Printf("Could not fetch blob entry for %s\n", name)
					} else {
						fmt.Printf("%s %x : packfile %x, offset %d, length %d\n",
							name,
							snapshot.Blob,
							snapshot.Location.Packfile,
							snapshot.Location.Offset,
							snapshot.Location.Length)
					}
				}
			}
			printDeleted := func(name string, Type resources.Type) {
				for deletedEntry, err := range st.ListDeletedResources(Type) {
					if err != nil {
						fmt.Printf("Could not fetch deleted blob entry for %s\n", name)
					} else {
						fmt.Printf("deleted %s: %x, when=%s\n",
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
	return 0, nil
}
