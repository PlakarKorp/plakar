package diag

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/PlakarKorp/kloset/btree"
	"github.com/PlakarKorp/kloset/locate"
	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/resources"
	"github.com/PlakarKorp/kloset/snapshot/vfs"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
)

type DiagXattr struct {
	subcommands.SubcommandBase

	SnapshotPath string
}

func (cmd *DiagXattr) Parse(ctx *appcontext.AppContext, args []string) error {
	flags := flag.NewFlagSet("diag xattr", flag.ExitOnError)
	flags.Parse(args)

	if len(flags.Args()) < 1 {
		return fmt.Errorf("usage: %s xattr SNAPSHOT[:PATH]", flags.Name())
	}

	cmd.RepositorySecret = ctx.GetSecret()
	cmd.SnapshotPath = flags.Args()[0]
	return nil
}

func (cmd *DiagXattr) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
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

	rd, err := repo.GetBlob(resources.RT_XATTR_BTREE, snap.Header.GetSource(0).VFS.Xattrs)
	if err != nil {
		return 1, err
	}

	store := repository.NewRepositoryStore[string, objects.MAC](repo, resources.RT_XATTR_NODE)
	tree, err := btree.Deserialize(rd, store, vfs.PathCmp)
	if err != nil {
		return 1, err
	}

	fs, err := snap.Filesystem()
	if err != nil {
		return 1, err
	}

	it, err := tree.ScanFrom(pathname)
	if err != nil {
		return 1, err
	}

	for it.Next() {
		path, xattrmac := it.Current()
		if !strings.HasPrefix(path, pathname) {
			break
		}

		xattr, err := fs.ResolveXattr(xattrmac)
		if err != nil {
			return 1, err
		}

		rd := vfs.NewObjectReader(repo, xattr.ResolvedObject, xattr.Size)
		value, err := io.ReadAll(rd)
		if err != nil {
			return 1, err
		}

		fmt.Println(xattr.Path, xattr.Name, string(value))
	}
	if err := it.Err(); err != nil {
		return 1, err
	}

	return 0, nil
}
