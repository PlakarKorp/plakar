package diag

import (
	"encoding/hex"
	"flag"
	"fmt"
	"io"

	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/resources"
	"github.com/PlakarKorp/plakar/appcontext"
)

func Object(ctx *appcontext.AppContext, repo *repository.Repository, args []string) error {
	flags := flag.NewFlagSet("diag objects", flag.ExitOnError)
	flags.Parse(args)

	if len(flags.Args()) < 1 {
		return fmt.Errorf("usage: %s object OBJECT", flags.Name())
	}

	objectID := flags.Args()[0]

	if len(objectID) != 64 {
		return fmt.Errorf("invalid object hash: %s", objectID)
	}

	b, err := hex.DecodeString(objectID)
	if err != nil {
		return fmt.Errorf("invalid object hash: %s", objectID)
	}

	// Convert the byte slice to a [32]byte
	var byteArray [32]byte
	copy(byteArray[:], b)

	rd, err := repo.GetBlob(resources.RT_OBJECT, byteArray)
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
	fmt.Fprintln(ctx.Stdout, "  chunks:")
	for _, chunk := range object.Chunks {
		fmt.Fprintf(ctx.Stdout, "    MAC: %x\n", chunk.ContentMAC)
	}
	return nil
}
