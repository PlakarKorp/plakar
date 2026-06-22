package diag

import (
	"encoding/hex"
	"flag"
	"fmt"
	"time"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
)

func Packfile(ctx *appcontext.AppContext, repo *repository.Repository, args []string) error {
	flags := flag.NewFlagSet("diag packfile", flag.ExitOnError)
	flags.Parse(args)

	if len(flags.Args()[0:]) == 0 {
		packfiles, err := repo.GetPackfiles()
		if err != nil {
			return err
		}

		for _, packfile := range packfiles {
			fmt.Fprintf(ctx.Stdout, "%x\n", packfile)
		}
	} else {
		for _, arg := range flags.Args()[0:] {
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

			p, err := repo.GetPackfile(byteArray)
			if err != nil {
				return err
			}

			fmt.Fprintf(ctx.Stdout, "Version: %s\n", p.Footer.Version)
			fmt.Fprintf(ctx.Stdout, "Timestamp: %s\n", time.Unix(0, p.Footer.Timestamp))
			fmt.Fprintf(ctx.Stdout, "Index MAC: %x\n", p.Footer.IndexMAC)
			fmt.Fprintln(ctx.Stdout)

			for i, entry := range p.Index {
				fmt.Fprintf(ctx.Stdout, "blob[%d]: %x %d %d %x %s\n", i, entry.MAC, entry.Offset, entry.Length, entry.Flags, entry.Type)
			}
		}
	}

	return nil
}
