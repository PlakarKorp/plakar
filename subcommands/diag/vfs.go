package diag

import (
	"flag"
	"fmt"
	"path"
	"time"

	"github.com/PlakarKorp/kloset/locate"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/resources"
	"github.com/PlakarKorp/kloset/snapshot/vfs"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/dustin/go-humanize"
)

func VFS(ctx *appcontext.AppContext, repo *repository.Repository, args []string) error {
	flags := flag.NewFlagSet("diag vfs", flag.ExitOnError)
	flags.Parse(args)

	if len(flags.Args()) < 1 {
		return fmt.Errorf("usage: %s vfs SNAPSHOT[:PATH]", flags.Name())
	}

	snap1, pathname, err := locate.OpenSnapshotByPath(repo, flags.Args()[0])
	if err != nil {
		return err
	}
	defer snap1.Close()

	fs, err := snap1.Filesystem()
	if err != nil {
		return err
	}

	pathname = path.Clean(pathname)
	entry, err := fs.GetEntry(pathname)
	if err != nil {
		return err
	}

	if entry.Stat().Mode().IsDir() {
		fmt.Fprintf(ctx.Stdout, "[DirEntry]\n")
	} else {
		fmt.Fprintf(ctx.Stdout, "[FileEntry]\n")
	}

	fmt.Fprintf(ctx.Stdout, "Version: %d\n", entry.Version)
	fmt.Fprintf(ctx.Stdout, "ParentPath: %s\n", entry.ParentPath)
	fmt.Fprintf(ctx.Stdout, "Name: %s\n", entry.Stat().Name())
	fmt.Fprintf(ctx.Stdout, "Size: %s (%d bytes)\n", humanize.IBytes(uint64(entry.Stat().Size())), entry.Stat().Size())
	fmt.Fprintf(ctx.Stdout, "Permissions: %s\n", entry.Stat().Mode())
	fmt.Fprintf(ctx.Stdout, "ModTime: %s\n", entry.Stat().ModTime())
	fmt.Fprintf(ctx.Stdout, "DeviceID: %d\n", entry.Stat().Dev())
	fmt.Fprintf(ctx.Stdout, "InodeID: %d\n", entry.Stat().Ino())
	fmt.Fprintf(ctx.Stdout, "UserID: %d\n", entry.Stat().Uid())
	fmt.Fprintf(ctx.Stdout, "GroupID: %d\n", entry.Stat().Gid())
	fmt.Fprintf(ctx.Stdout, "Username: %s\n", entry.Stat().Username())
	fmt.Fprintf(ctx.Stdout, "Groupname: %s\n", entry.Stat().Groupname())
	fmt.Fprintf(ctx.Stdout, "NumLinks: %d\n", entry.Stat().Nlink())
	fmt.Fprintf(ctx.Stdout, "ExtendedAttributes: %s\n", entry.ExtendedAttributes)
	fmt.Fprintf(ctx.Stdout, "FileAttributes: %v\n", entry.FileAttributes)
	if entry.SymlinkTarget != "" {
		fmt.Fprintf(ctx.Stdout, "SymlinkTarget: %s\n", entry.SymlinkTarget)
	}
	fmt.Fprintf(ctx.Stdout, "Classification:\n")
	for _, classification := range entry.Classifications {
		fmt.Fprintf(ctx.Stdout, " - %s:\n", classification.Analyzer)
		for _, class := range classification.Classes {
			fmt.Fprintf(ctx.Stdout, "   - %s\n", class)
		}
	}
	fmt.Fprintf(ctx.Stdout, "CustomMetadata: %s\n", entry.CustomMetadata)
	fmt.Fprintf(ctx.Stdout, "Tags: %s\n", entry.Tags)
	fmt.Fprintf(ctx.Stdout, "ExtendedAttributes: %v\n", entry.ExtendedAttributes)

	summary := entry.Summary
	if summary == nil && entry.IsDir() {
		tree, err := snap1.SummaryIdx()
		if err != nil {
			return err
		}

		key, found, err := tree.Find(pathname)
		if err != nil {
			return err
		}
		if !found {
			return fmt.Errorf("could not resolve pathname: %s", pathname)
		}

		serializedSummary, err := repo.GetBlobBytes(resources.RT_VFS_SUMMARY, key)
		if err != nil {
			return err
		}

		summary, err = vfs.SummaryFromBytes(serializedSummary)
		if err != nil {
			return err
		}
	}

	if summary != nil {
		fmt.Fprintf(ctx.Stdout, "Below.Directories: %d\n", summary.Below.Directories)
		fmt.Fprintf(ctx.Stdout, "Below.Files: %d\n", summary.Below.Files)
		fmt.Fprintf(ctx.Stdout, "Below.Symlinks: %d\n", summary.Below.Symlinks)
		fmt.Fprintf(ctx.Stdout, "Below.Devices: %d\n", summary.Below.Devices)
		fmt.Fprintf(ctx.Stdout, "Below.Pipes: %d\n", summary.Below.Pipes)
		fmt.Fprintf(ctx.Stdout, "Below.Sockets: %d\n", summary.Below.Sockets)
		fmt.Fprintf(ctx.Stdout, "Below.Setuid: %d\n", summary.Below.Setuid)
		fmt.Fprintf(ctx.Stdout, "Below.Setgid: %d\n", summary.Below.Setgid)
		fmt.Fprintf(ctx.Stdout, "Below.Sticky: %d\n", summary.Below.Sticky)
		fmt.Fprintf(ctx.Stdout, "Below.Objects: %d\n", summary.Below.Objects)
		fmt.Fprintf(ctx.Stdout, "Below.Chunks: %d\n", summary.Below.Chunks)
		fmt.Fprintf(ctx.Stdout, "Below.MinSize: %s (%d bytes)\n", humanize.IBytes(uint64(summary.Below.MinSize)), summary.Below.MinSize)
		fmt.Fprintf(ctx.Stdout, "Below.MaxSize: %s (%d bytes)\n", humanize.IBytes(uint64(summary.Below.MaxSize)), summary.Below.MaxSize)
		fmt.Fprintf(ctx.Stdout, "Below.Size: %s (%d bytes)\n", humanize.IBytes(uint64(summary.Below.Size)), summary.Below.Size)
		fmt.Fprintf(ctx.Stdout, "Below.MinModTime: %s\n", time.Unix(summary.Below.MinModTime, 0))
		fmt.Fprintf(ctx.Stdout, "Below.MaxModTime: %s\n", time.Unix(summary.Below.MaxModTime, 0))
		fmt.Fprintf(ctx.Stdout, "Below.MinEntropy: %f\n", summary.Below.MinEntropy)
		fmt.Fprintf(ctx.Stdout, "Below.MaxEntropy: %f\n", summary.Below.MaxEntropy)
		fmt.Fprintf(ctx.Stdout, "Below.HiEntropy: %d\n", summary.Below.HiEntropy)
		fmt.Fprintf(ctx.Stdout, "Below.LoEntropy: %d\n", summary.Below.LoEntropy)
		fmt.Fprintf(ctx.Stdout, "Below.MIMEAudio: %d\n", summary.Below.MIMEAudio)
		fmt.Fprintf(ctx.Stdout, "Below.MIMEVideo: %d\n", summary.Below.MIMEVideo)
		fmt.Fprintf(ctx.Stdout, "Below.MIMEImage: %d\n", summary.Below.MIMEImage)
		fmt.Fprintf(ctx.Stdout, "Below.MIMEText: %d\n", summary.Below.MIMEText)
		fmt.Fprintf(ctx.Stdout, "Below.MIMEApplication: %d\n", summary.Below.MIMEApplication)
		fmt.Fprintf(ctx.Stdout, "Below.MIMEOther: %d\n", summary.Below.MIMEOther)
		fmt.Fprintf(ctx.Stdout, "Below.Errors: %d\n", summary.Below.Errors)
		fmt.Fprintf(ctx.Stdout, "Directory.Directories: %d\n", summary.Directory.Directories)
		fmt.Fprintf(ctx.Stdout, "Directory.Files: %d\n", summary.Directory.Files)
		fmt.Fprintf(ctx.Stdout, "Directory.Symlinks: %d\n", summary.Directory.Symlinks)
		fmt.Fprintf(ctx.Stdout, "Directory.Devices: %d\n", summary.Directory.Devices)
		fmt.Fprintf(ctx.Stdout, "Directory.Pipes: %d\n", summary.Directory.Pipes)
		fmt.Fprintf(ctx.Stdout, "Directory.Sockets: %d\n", summary.Directory.Sockets)
		fmt.Fprintf(ctx.Stdout, "Directory.Setuid: %d\n", summary.Directory.Setuid)
		fmt.Fprintf(ctx.Stdout, "Directory.Setgid: %d\n", summary.Directory.Setgid)
		fmt.Fprintf(ctx.Stdout, "Directory.Sticky: %d\n", summary.Directory.Sticky)
		fmt.Fprintf(ctx.Stdout, "Directory.Objects: %d\n", summary.Directory.Objects)
		fmt.Fprintf(ctx.Stdout, "Directory.Chunks: %d\n", summary.Directory.Chunks)
		fmt.Fprintf(ctx.Stdout, "Directory.MinSize: %s (%d bytes)\n", humanize.IBytes(uint64(summary.Directory.MinSize)), summary.Directory.MinSize)
		fmt.Fprintf(ctx.Stdout, "Directory.MaxSize: %s (%d bytes)\n", humanize.IBytes(uint64(summary.Directory.MaxSize)), summary.Directory.MaxSize)
		fmt.Fprintf(ctx.Stdout, "Directory.Size: %s (%d bytes)\n", humanize.IBytes(uint64(summary.Directory.Size)), summary.Directory.Size)
		fmt.Fprintf(ctx.Stdout, "Directory.MinModTime: %s\n", time.Unix(summary.Directory.MinModTime, 0))
		fmt.Fprintf(ctx.Stdout, "Directory.MaxModTime: %s\n", time.Unix(summary.Directory.MaxModTime, 0))
		fmt.Fprintf(ctx.Stdout, "Directory.MinEntropy: %f\n", summary.Directory.MinEntropy)
		fmt.Fprintf(ctx.Stdout, "Directory.MaxEntropy: %f\n", summary.Directory.MaxEntropy)
		fmt.Fprintf(ctx.Stdout, "Directory.AvgEntropy: %f\n", summary.Directory.AvgEntropy)
		fmt.Fprintf(ctx.Stdout, "Directory.HiEntropy: %d\n", summary.Directory.HiEntropy)
		fmt.Fprintf(ctx.Stdout, "Directory.LoEntropy: %d\n", summary.Directory.LoEntropy)
		fmt.Fprintf(ctx.Stdout, "Directory.MIMEAudio: %d\n", summary.Directory.MIMEAudio)
		fmt.Fprintf(ctx.Stdout, "Directory.MIMEVideo: %d\n", summary.Directory.MIMEVideo)
		fmt.Fprintf(ctx.Stdout, "Directory.MIMEImage: %d\n", summary.Directory.MIMEImage)
		fmt.Fprintf(ctx.Stdout, "Directory.MIMEText: %d\n", summary.Directory.MIMEText)
		fmt.Fprintf(ctx.Stdout, "Directory.MIMEApplication: %d\n", summary.Directory.MIMEApplication)
		fmt.Fprintf(ctx.Stdout, "Directory.MIMEOther: %d\n", summary.Directory.MIMEOther)
		fmt.Fprintf(ctx.Stdout, "Directory.Errors: %d\n", summary.Directory.Errors)
		fmt.Fprintf(ctx.Stdout, "Directory.Children: %d\n", summary.Directory.Children)
	}

	if entry.IsDir() {
		iter, err := entry.Getdents(fs)
		if err != nil {
			return err
		}
		offset := 0
		for child := range iter {
			fmt.Fprintf(ctx.Stdout, "Child[%d].FileInfo.Name(): %s\n", offset, child.Stat().Name())
			fmt.Fprintf(ctx.Stdout, "Child[%d].FileInfo.Size(): %d\n", offset, child.Stat().Size())
			fmt.Fprintf(ctx.Stdout, "Child[%d].FileInfo.Mode(): %s\n", offset, child.Stat().Mode())
			fmt.Fprintf(ctx.Stdout, "Child[%d].FileInfo.Dev(): %d\n", offset, child.Stat().Dev())
			fmt.Fprintf(ctx.Stdout, "Child[%d].FileInfo.Ino(): %d\n", offset, child.Stat().Ino())
			fmt.Fprintf(ctx.Stdout, "Child[%d].FileInfo.Uid(): %d\n", offset, child.Stat().Uid())
			fmt.Fprintf(ctx.Stdout, "Child[%d].FileInfo.Gid(): %d\n", offset, child.Stat().Gid())
			fmt.Fprintf(ctx.Stdout, "Child[%d].FileInfo.Username(): %s\n", offset, child.Stat().Username())
			fmt.Fprintf(ctx.Stdout, "Child[%d].FileInfo.Groupname(): %s\n", offset, child.Stat().Groupname())
			fmt.Fprintf(ctx.Stdout, "Child[%d].FileInfo.Nlink(): %d\n", offset, child.Stat().Nlink())
			fmt.Fprintf(ctx.Stdout, "Child[%d].ExtendedAttributes(): %v\n", offset, child.ExtendedAttributes)
			offset++
		}
	}

	offset := 0
	for entry, err := range fs.Errors(pathname) {
		if err != nil {
			return fmt.Errorf("failure in scanning errors: %w", err)
		}
		fmt.Fprintf(ctx.Stdout, "Error[%d]: %s: %s\n", offset, entry.Name, entry.Error)
		offset++
	}
	return nil
}
