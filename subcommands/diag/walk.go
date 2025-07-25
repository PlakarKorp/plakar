package diag

import (
	"flag"
	"fmt"
	"log"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot/vfs"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/PlakarKorp/plakar/utils"
)

type DiagWalk struct {
	subcommands.SubcommandBase

	SnapshotPath string
}

func (cmd *DiagWalk) Parse(ctx *appcontext.AppContext, args []string) error {
	flags := flag.NewFlagSet("diag vfs", flag.ExitOnError)
	flags.Parse(args)

	if len(flags.Args()) < 1 {
		return fmt.Errorf("usage: %s vfs SNAPSHOT[:PATH]", flags.Name())
	}

	cmd.RepositorySecret = ctx.GetSecret()
	cmd.SnapshotPath = flags.Args()[0]

	return nil
}

func (cmd *DiagWalk) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	snap1, pathname, err := utils.OpenSnapshotByPath(repo, cmd.SnapshotPath)
	if err != nil {
		return 1, err
	}
	defer snap1.Close()

	fs, err := snap1.Filesystem()
	if err != nil {
		return 1, err
	}

	var (
		files int
		dirs  int
	)

	fs.WalkDir(pathname, func(path string, entry *vfs.Entry, err error) error {
		log.Print("dirs\t", dirs, " files\t", files, "\n")

		if err != nil {
			log.Println("failure:", err)
			return err
		}

		// if entry.IsDir() {
		// 	dirs++
		// } else {
		// 	files++
		// }
		files++

		if err := ctx.Err(); err != nil {
			log.Println("cancelled!")
			return err
		}
		return nil
	})

	log.Print("dirs\t", dirs, "\n")
	log.Print("files\t", files, "\n")

	return 0, nil
}
