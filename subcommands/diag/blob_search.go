package diag

import (
	"encoding/hex"
	"flag"
	"fmt"
	"io"

	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/repository/state"
	"github.com/PlakarKorp/kloset/resources"
	"github.com/PlakarKorp/plakar/appcontext"
)

func BlobSearch(ctx *appcontext.AppContext, repo *repository.Repository, args []string) error {
	flags := flag.NewFlagSet("diag packfile", flag.ExitOnError)
	flags.Parse(args)

	if len(flags.Args()) < 1 {
		return fmt.Errorf("usage: %s blobsearch OBJECT", flags.Name())
	}

	objectID := flags.Args()[0]

	fmt.Fprintf(ctx.Stdout, "Warning this command is slow and expensive. Use with caution.\n")

	if len(objectID) != 64 {
		return fmt.Errorf("invalid object hash: %s", objectID)
	}

	b, err := hex.DecodeString(objectID)
	if err != nil {
		return fmt.Errorf("invalid object hash: %s", objectID)
	}

	needleMAC := objects.MAC(b)

	packfiles, err := repo.GetPackfiles()
	if err != nil {
		return err
	}

	for _, packfileMac := range packfiles {
		p, err := repo.GetPackfile(packfileMac)
		if err != nil {
			return err
		}

		for _, entry := range p.Index {
			if entry.MAC == needleMAC {
				fmt.Fprintf(ctx.Stdout, "Found candidate [%x] in packfile [%x] at : %d %d %s\n", entry.MAC, packfileMac, entry.Offset, entry.Length, entry.Type)
				if entry.Type == resources.RT_OBJECT {
					rd, err := repo.GetPackfileBlob(state.Location{Packfile: packfileMac, Offset: entry.Offset, Length: entry.Length})
					if err != nil {
						return err
					}

					blob, err := io.ReadAll(rd)
					if err != nil {
						return err
					}

					object, err := objects.NewObjectFromBytes(blob)
					if err != nil {
						return err
					}

					fmt.Fprintf(ctx.Stdout, "object: %x\n", object.ContentMAC)
					fmt.Fprintln(ctx.Stdout, "  type:", object.ContentType)
				}
			}
		}
	}

	return nil
}
