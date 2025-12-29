package diag

import (
	"flag"
	"fmt"
	"path"
	"time"

	"github.com/PlakarKorp/kloset/locate"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/dustin/go-humanize"
)

type DiagVFS struct {
	subcommands.SubcommandBase

	SnapshotPath string
}

func (cmd *DiagVFS) Parse(ctx *appcontext.AppContext, args []string) error {
	flags := flag.NewFlagSet("diag vfs", flag.ExitOnError)
	flags.Parse(args)

	if len(flags.Args()) < 1 {
		return fmt.Errorf("usage: %s vfs SNAPSHOT[:PATH]", flags.Name())
	}

	cmd.RepositorySecret = ctx.GetSecret()
	cmd.SnapshotPath = flags.Args()[0]

	return nil
}

func (cmd *DiagVFS) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	snap1, pathname, err := locate.OpenSnapshotByPath(repo, cmd.SnapshotPath)
	if err != nil {
		return 1, err
	}
	defer snap1.Close()

	fs, err := snap1.Filesystem()
	if err != nil {
		return 1, err
	}

	pathname = path.Clean(pathname)
	entry, err := fs.GetEntry(pathname)
	if err != nil {
		return 1, err
	}

	if entry.Stat().Mode().IsDir() {
		fmt.Printf("[DirEntry]\n")
	} else {
		fmt.Printf("[FileEntry]\n")
	}

	fmt.Printf("Version: %d\n", entry.Version)
	fmt.Printf("ParentPath: %s\n", entry.ParentPath)
	fmt.Printf("Name: %s\n", entry.Stat().Name())
	fmt.Printf("Size: %s (%d bytes)\n", humanize.IBytes(uint64(entry.Stat().Size())), entry.Stat().Size())
	fmt.Printf("Permissions: %s\n", entry.Stat().Mode())
	fmt.Printf("ModTime: %s\n", entry.Stat().ModTime())
	fmt.Printf("DeviceID: %d\n", entry.Stat().Dev())
	fmt.Printf("InodeID: %d\n", entry.Stat().Ino())
	fmt.Printf("UserID: %d\n", entry.Stat().Uid())
	fmt.Printf("GroupID: %d\n", entry.Stat().Gid())
	fmt.Printf("Username: %s\n", entry.Stat().Username())
	fmt.Printf("Groupname: %s\n", entry.Stat().Groupname())
	fmt.Printf("NumLinks: %d\n", entry.Stat().Nlink())
	fmt.Printf("ExtendedAttributes: %s\n", entry.ExtendedAttributes)
	fmt.Printf("FileAttributes: %v\n", entry.FileAttributes)
	if entry.SymlinkTarget != "" {
		fmt.Printf("SymlinkTarget: %s\n", entry.SymlinkTarget)
	}
	fmt.Printf("Classification:\n")
	for _, classification := range entry.Classifications {
		fmt.Printf(" - %s:\n", classification.Analyzer)
		for _, class := range classification.Classes {
			fmt.Printf("   - %s\n", class)
		}
	}
	fmt.Printf("CustomMetadata: %s\n", entry.CustomMetadata)
	fmt.Printf("Tags: %s\n", entry.Tags)
	fmt.Printf("ExtendedAttributes: %v\n", entry.ExtendedAttributes)

	if entry.Summary != nil {
		fmt.Printf("Below.Directories: %d\n", entry.Summary.Below.Directories)
		fmt.Printf("Below.Files: %d\n", entry.Summary.Below.Files)
		fmt.Printf("Below.Symlinks: %d\n", entry.Summary.Below.Symlinks)
		fmt.Printf("Below.Devices: %d\n", entry.Summary.Below.Devices)
		fmt.Printf("Below.Pipes: %d\n", entry.Summary.Below.Pipes)
		fmt.Printf("Below.Sockets: %d\n", entry.Summary.Below.Sockets)
		fmt.Printf("Below.Setuid: %d\n", entry.Summary.Below.Setuid)
		fmt.Printf("Below.Setgid: %d\n", entry.Summary.Below.Setgid)
		fmt.Printf("Below.Sticky: %d\n", entry.Summary.Below.Sticky)
		fmt.Printf("Below.Objects: %d\n", entry.Summary.Below.Objects)
		fmt.Printf("Below.Chunks: %d\n", entry.Summary.Below.Chunks)
		fmt.Printf("Below.MinSize: %s (%d bytes)\n", humanize.IBytes(uint64(entry.Summary.Below.MinSize)), entry.Summary.Below.MinSize)
		fmt.Printf("Below.MaxSize: %s (%d bytes)\n", humanize.IBytes(uint64(entry.Summary.Below.MaxSize)), entry.Summary.Below.MaxSize)
		fmt.Printf("Below.Size: %s (%d bytes)\n", humanize.IBytes(uint64(entry.Summary.Below.Size)), entry.Summary.Below.Size)
		fmt.Printf("Below.MinModTime: %s\n", time.Unix(entry.Summary.Below.MinModTime, 0))
		fmt.Printf("Below.MaxModTime: %s\n", time.Unix(entry.Summary.Below.MaxModTime, 0))
		fmt.Printf("Below.MinEntropy: %f\n", entry.Summary.Below.MinEntropy)
		fmt.Printf("Below.MaxEntropy: %f\n", entry.Summary.Below.MaxEntropy)
		fmt.Printf("Below.HiEntropy: %d\n", entry.Summary.Below.HiEntropy)
		fmt.Printf("Below.LoEntropy: %d\n", entry.Summary.Below.LoEntropy)
		fmt.Printf("Below.MIMEAudio: %d\n", entry.Summary.Below.MIMEAudio)
		fmt.Printf("Below.MIMEVideo: %d\n", entry.Summary.Below.MIMEVideo)
		fmt.Printf("Below.MIMEImage: %d\n", entry.Summary.Below.MIMEImage)
		fmt.Printf("Below.MIMEText: %d\n", entry.Summary.Below.MIMEText)
		fmt.Printf("Below.MIMEApplication: %d\n", entry.Summary.Below.MIMEApplication)
		fmt.Printf("Below.MIMEOther: %d\n", entry.Summary.Below.MIMEOther)
		fmt.Printf("Below.Errors: %d\n", entry.Summary.Below.Errors)
		fmt.Printf("Directory.Directories: %d\n", entry.Summary.Directory.Directories)
		fmt.Printf("Directory.Files: %d\n", entry.Summary.Directory.Files)
		fmt.Printf("Directory.Symlinks: %d\n", entry.Summary.Directory.Symlinks)
		fmt.Printf("Directory.Devices: %d\n", entry.Summary.Directory.Devices)
		fmt.Printf("Directory.Pipes: %d\n", entry.Summary.Directory.Pipes)
		fmt.Printf("Directory.Sockets: %d\n", entry.Summary.Directory.Sockets)
		fmt.Printf("Directory.Setuid: %d\n", entry.Summary.Directory.Setuid)
		fmt.Printf("Directory.Setgid: %d\n", entry.Summary.Directory.Setgid)
		fmt.Printf("Directory.Sticky: %d\n", entry.Summary.Directory.Sticky)
		fmt.Printf("Directory.Objects: %d\n", entry.Summary.Directory.Objects)
		fmt.Printf("Directory.Chunks: %d\n", entry.Summary.Directory.Chunks)
		fmt.Printf("Directory.MinSize: %s (%d bytes)\n", humanize.IBytes(uint64(entry.Summary.Directory.MinSize)), entry.Summary.Directory.MinSize)
		fmt.Printf("Directory.MaxSize: %s (%d bytes)\n", humanize.IBytes(uint64(entry.Summary.Directory.MaxSize)), entry.Summary.Directory.MaxSize)
		fmt.Printf("Directory.Size: %s (%d bytes)\n", humanize.IBytes(uint64(entry.Summary.Directory.Size)), entry.Summary.Directory.Size)
		fmt.Printf("Directory.MinModTime: %s\n", time.Unix(entry.Summary.Directory.MinModTime, 0))
		fmt.Printf("Directory.MaxModTime: %s\n", time.Unix(entry.Summary.Directory.MaxModTime, 0))
		fmt.Printf("Directory.MinEntropy: %f\n", entry.Summary.Directory.MinEntropy)
		fmt.Printf("Directory.MaxEntropy: %f\n", entry.Summary.Directory.MaxEntropy)
		fmt.Printf("Directory.AvgEntropy: %f\n", entry.Summary.Directory.AvgEntropy)
		fmt.Printf("Directory.HiEntropy: %d\n", entry.Summary.Directory.HiEntropy)
		fmt.Printf("Directory.LoEntropy: %d\n", entry.Summary.Directory.LoEntropy)
		fmt.Printf("Directory.MIMEAudio: %d\n", entry.Summary.Directory.MIMEAudio)
		fmt.Printf("Directory.MIMEVideo: %d\n", entry.Summary.Directory.MIMEVideo)
		fmt.Printf("Directory.MIMEImage: %d\n", entry.Summary.Directory.MIMEImage)
		fmt.Printf("Directory.MIMEText: %d\n", entry.Summary.Directory.MIMEText)
		fmt.Printf("Directory.MIMEApplication: %d\n", entry.Summary.Directory.MIMEApplication)
		fmt.Printf("Directory.MIMEOther: %d\n", entry.Summary.Directory.MIMEOther)
		fmt.Printf("Directory.Errors: %d\n", entry.Summary.Directory.Errors)
		fmt.Printf("Directory.Children: %d\n", entry.Summary.Directory.Children)
	}

	if entry.IsDir() {
		iter, err := entry.Getdents(fs)
		if err != nil {
			return 1, err
		}
		offset := 0
		for child := range iter {
			fmt.Printf("Child[%d].FileInfo.Name(): %s\n", offset, child.Stat().Name())
			fmt.Printf("Child[%d].FileInfo.Size(): %d\n", offset, child.Stat().Size())
			fmt.Printf("Child[%d].FileInfo.Mode(): %s\n", offset, child.Stat().Mode())
			fmt.Printf("Child[%d].FileInfo.Dev(): %d\n", offset, child.Stat().Dev())
			fmt.Printf("Child[%d].FileInfo.Ino(): %d\n", offset, child.Stat().Ino())
			fmt.Printf("Child[%d].FileInfo.Uid(): %d\n", offset, child.Stat().Uid())
			fmt.Printf("Child[%d].FileInfo.Gid(): %d\n", offset, child.Stat().Gid())
			fmt.Printf("Child[%d].FileInfo.Username(): %s\n", offset, child.Stat().Username())
			fmt.Printf("Child[%d].FileInfo.Groupname(): %s\n", offset, child.Stat().Groupname())
			fmt.Printf("Child[%d].FileInfo.Nlink(): %d\n", offset, child.Stat().Nlink())
			fmt.Printf("Child[%d].ExtendedAttributes(): %v\n", offset, child.ExtendedAttributes)
			offset++
		}
	}

	errors, err := fs.Errors(pathname)
	if err != nil {
		return 1, err
	}
	offset := 0
	for err := range errors {
		fmt.Printf("Error[%d]: %s: %s\n", offset, err.Name, err.Error)
		offset++
	}
	return 0, nil
}
