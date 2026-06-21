package diag

import (
	"encoding/hex"
	"flag"
	"fmt"

	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/resources"
	"github.com/PlakarKorp/kloset/snapshot/header"
	"github.com/PlakarKorp/kloset/snapshot/vfs"
	"github.com/PlakarKorp/plakar/appcontext"
)

func Blob(ctx *appcontext.AppContext, repo *repository.Repository, args []string) error {
	flags := flag.NewFlagSet("diag blob", flag.ExitOnError)
	flags.Parse(args)

	if len(flags.Args()) != 2 {
		return fmt.Errorf("usage: %s blob type mac", flags.Name())
	}

	blobtype, err := resources.FromString(flags.Arg(0))
	if err != nil {
		return fmt.Errorf("%w: %s", err, blobtype)
	}

	macbytes, err := hex.DecodeString(flags.Arg(1))
	if err != nil {
		return fmt.Errorf("%w: %s", err, flags.Arg(1))
	}

	if len(macbytes) != 32 {
		return fmt.Errorf("invalid length for the mac: %s", flags.Arg(1))
	}

	mac := objects.MAC(macbytes)

	buf, err := repo.GetBlobBytes(blobtype, mac)
	if err != nil {
		return fmt.Errorf("failed to open blob %s %x: %w", blobtype, mac, err)
	}

	switch blobtype {
	case resources.RT_SNAPSHOT:
		hdr, err := header.NewFromBytes(buf)
		if err != nil {
			return fmt.Errorf("failed to deserialize %s %x: %w",
				blobtype, mac, err)
		}
		fmt.Fprintf(ctx.Stdout, "%+v\n", hdr)

	case resources.RT_OBJECT:
		obj, err := objects.NewObjectFromBytes(buf)
		if err != nil {
			return fmt.Errorf("failed to deserialize %s %x: %w",
				blobtype, mac, err)
		}
		fmt.Fprintf(ctx.Stdout, "%+v\n", obj)

	case resources.RT_CHUNK:
		chunk, err := objects.NewChunkFromBytes(buf)
		if err != nil {
			return fmt.Errorf("failed to deserialize %s %x: %w",
				blobtype, mac, err)
		}
		fmt.Fprintf(ctx.Stdout, "%+v\n", chunk)

	case resources.RT_VFS_ENTRY:
		entry, err := vfs.EntryFromBytes(buf)
		if err != nil {
			return fmt.Errorf("failed to deserialize %s %x: %w",
				blobtype, mac, err)
		}
		fmt.Fprintf(ctx.Stdout, "%+v\n", entry)

	case resources.RT_ERROR_ENTRY:
		error, err := vfs.ErrorItemFromBytes(buf)
		if err != nil {
			return fmt.Errorf("failed to deserialize %s %x: %w",
				blobtype, mac, err)
		}
		fmt.Fprintf(ctx.Stdout, "%+v\n", error)

	case resources.RT_XATTR_ENTRY:
		xattr, err := vfs.XattrFromBytes(buf)
		if err != nil {
			return fmt.Errorf("failed to deserialize %s %x: %w",
				blobtype, mac, err)
		}
		fmt.Fprintf(ctx.Stdout, "%+v\n", xattr)

	default:
		return fmt.Errorf("don't know how to deserialize %s", blobtype)
	}

	return nil
}
