package diag

import (
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/PlakarKorp/kloset/locate"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/kloset/snapshot/vfs"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/vmihailenco/msgpack/v5"
)

type DiagDirPack struct {
	subcommands.SubcommandBase

	SnapshotPath string
}

func (cmd *DiagDirPack) Parse(ctx *appcontext.AppContext, args []string) error {
	flags := flag.NewFlagSet("diag dirpack", flag.ExitOnError)
	flags.Parse(args)

	if len(flags.Args()) < 1 {
		return fmt.Errorf("usage: %s dirpack SNAPSHOT[:PATH]", flags.Name())
	}

	cmd.RepositorySecret = ctx.GetSecret()
	cmd.SnapshotPath = flags.Args()[0]

	return nil
}

func readDirPackHdr(rd io.Reader) (typ snapshot.DirPackEntry, siz uint32, err error) {
	endian := binary.LittleEndian
	if err = binary.Read(rd, endian, &typ); err != nil {
		return
	}
	if err = binary.Read(rd, endian, &siz); err != nil {
		return
	}
	return
}

func (cmd *DiagDirPack) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	snap, pathname, err := locate.OpenSnapshotByPath(repo, cmd.SnapshotPath)
	if err != nil {
		return 1, err
	}
	defer snap.Close()

	if pathname == "" {
		pathname = "/"
	}
	if !strings.HasSuffix(pathname, "/") {
		pathname += "/"
	}

	tree, err := snap.DirPack()
	if err != nil {
		return 1, err
	}
	if tree == nil {
		return 1, fmt.Errorf("no dirpack index available in the snapshot")
	}

	it, err := tree.ScanFrom(pathname)
	if err != nil {
		return 1, err
	}

	for it.Next() {
		fmt.Println("===============================================")
		path, dirpackmac := it.Current()
		if !strings.HasPrefix(path, pathname) {
			break
		}

		fmt.Fprintf(ctx.Stdout, "%s %x\n", path, dirpackmac)

		obj, err := snap.LookupObject(dirpackmac)
		if err != nil {
			return 1, fmt.Errorf("failed to get object %x: %w", dirpackmac, err)
		}

		var size int64
		for _, c := range obj.Chunks {
			size += int64(c.Length)
		}

		rd := vfs.NewObjectReader(repo, obj, size)

		for {
			typ, siz, err := readDirPackHdr(rd)
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				return 1, fmt.Errorf("failed to read: %w", err)
			}

			var entry vfs.Entry
			lrd := io.LimitReader(rd, int64(siz-uint32(len(entry.MAC[:]))))
			err = msgpack.NewDecoder(lrd).Decode(&entry)
			if err != nil {
				return 1, fmt.Errorf("failed to read entry: %w", err)
			}

			if _, err := io.ReadFull(rd, entry.MAC[:]); err != nil {
				return 1, fmt.Errorf("failed to read entry mac: %w", err)
			}

			fmt.Fprintf(ctx.Stdout, "vfs-entry %x %s %v %v %s\n", entry.MAC, path,
				siz, typ, entry.Name())
		}
	}
	if err := it.Err(); err != nil {
		return 1, err
	}

	return 0, nil
}
