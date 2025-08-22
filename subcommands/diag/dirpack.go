package diag

import (
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/kloset/snapshot/vfs"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/locate"
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

	obj, err := snap.DirPackObject()
	if err != nil {
		return 1, err
	}

	var size int64
	for _, ch := range obj.Chunks {
		size += int64(ch.Length)
	}

	rd := vfs.NewObjectReader(repo, obj, size)
	for it.Next() {
		fmt.Println("===============================================")
		path, pair := it.Current()
		if !strings.HasPrefix(path, pathname) {
			break
		}

		offset := (pair >> 32)
		length := pair & 0xFFFFFFFF

		fmt.Fprintf(ctx.Stdout, "Offset %d, Length %d\n", offset, length)
		rd.Seek(int64(offset), io.SeekStart)
		lrd := io.LimitReader(rd, int64(length))

		for {
			typ, siz, err := readDirPackHdr(lrd)
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				return 1, fmt.Errorf("failed to read: %w", err)
			}

			llrd := io.LimitReader(lrd, int64(siz))

			var entry vfs.Entry
			err = msgpack.NewDecoder(llrd).Decode(&entry)
			if err != nil {
				return 1, fmt.Errorf("failed to read entry: %w", err)
			}

			fmt.Fprintln(ctx.Stdout, path, siz, typ, entry.Name())
		}
	}
	if err := it.Err(); err != nil {
		return 1, err
	}

	return 0, nil
}
