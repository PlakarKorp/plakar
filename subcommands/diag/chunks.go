package diag

import (
	"flag"
	"fmt"

	"github.com/PlakarKorp/kloset/locate"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
)

func Chunks(ctx *appcontext.AppContext, repo *repository.Repository, args []string) error {
	flags := flag.NewFlagSet("diag chunks", flag.ExitOnError)
	flags.Parse(args)

	if len(flags.Args()) != 1 {
		return fmt.Errorf("usage: %s chunks SNAPSHOT:PATH", flags.Name())
	}

	snap, pathname, err := locate.OpenSnapshotByPath(repo, flags.Args()[0])
	if err != nil {
		return err
	}
	defer snap.Close()

	fs, err := snap.Filesystem()
	if err != nil {
		return err
	}

	entry, err := fs.GetEntry(pathname)
	if err != nil {
		return err
	}

	if entry.ResolvedObject == nil {
		return fmt.Errorf("no object for path: %s", pathname)
	}

	var offset int64
	for i, chunk := range entry.ResolvedObject.Chunks {
		fmt.Fprintf(ctx.Stdout, "Chunk[%d]: offset=%d length=%d mac=%x entropy=%f\n",
			i, offset, chunk.Length, chunk.ContentMAC, chunk.Entropy)
		offset += int64(chunk.Length)
	}

	return nil
}
