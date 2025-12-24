package diag

import (
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
)

type DiagPackfile struct {
	subcommands.SubcommandBase

	Args []string
}

func (cmd *DiagPackfile) Parse(ctx *appcontext.AppContext, args []string) error {
	flags := flag.NewFlagSet("diag packfile", flag.ExitOnError)
	flags.Parse(args)

	cmd.RepositorySecret = ctx.GetSecret()
	cmd.Args = flags.Args()[0:]

	return nil
}

func (cmd *DiagPackfile) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	if len(cmd.Args) == 0 {
		packfiles, err := repo.GetPackfiles()
		if err != nil {
			return 1, err
		}

		for _, packfile := range packfiles {
			fmt.Printf("%x\n", packfile)
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

			p, err := repo.GetPackfile(byteArray)
			if err != nil {
				return 1, err
			}

			fmt.Printf("Version: %s\n", p.Footer.Version)
			fmt.Printf("Timestamp: %s\n", time.Unix(0, p.Footer.Timestamp))
			fmt.Printf("Index MAC: %x\n", p.Footer.IndexMAC)
			fmt.Fprintln(os.Stdout)

			for i, entry := range p.Index {
				fmt.Printf("blob[%d]: %x %d %d %x %s\n", i, entry.MAC, entry.Offset, entry.Length, entry.Flags, entry.Type)
			}
		}
	}
	return 0, nil
}
