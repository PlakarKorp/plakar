package info

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/PlakarKorp/kloset/locate"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/dustin/go-humanize"
	"github.com/google/uuid"
)

func (cmd *Info) executeSnapshot(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	snap, _, err := locate.OpenSnapshotByPath(repo, cmd.SnapshotID)
	if err != nil {
		return 1, err
	}
	defer snap.Close()

	header := snap.Header

	indexID := header.GetIndexID()
	fmt.Printf("Version: %s\n", repo.Configuration().Version)
	fmt.Printf("SnapshotID: %s\n", hex.EncodeToString(indexID[:]))
	fmt.Printf("Timestamp: %s\n", header.Timestamp)
	fmt.Printf("Duration: %s\n", header.Duration)

	fmt.Printf("Name: %s\n", header.Name)
	fmt.Printf("Environment: %s\n", header.Environment)
	fmt.Printf("Perimeter: %s\n", header.Perimeter)
	fmt.Printf("Category: %s\n", header.Category)
	if len(header.Tags) > 0 {
		fmt.Printf("Tags: %s\n", strings.Join(header.Tags, ", "))
	}

	if header.Identity.Identifier != uuid.Nil {
		fmt.Println("Identity:")
		fmt.Printf(" - Identifier: %s\n", header.Identity.Identifier)
		fmt.Printf(" - PublicKey: %s\n", base64.RawStdEncoding.EncodeToString(header.Identity.PublicKey))
	}

	fmt.Printf("VFS: %x\n", header.GetSource(0).VFS)

	fmt.Println("Importer:")
	fmt.Printf(" - Type: %s\n", header.GetSource(0).Importer.Type)
	fmt.Printf(" - Origin: %s\n", header.GetSource(0).Importer.Origin)
	fmt.Printf(" - Directory: %s\n", header.GetSource(0).Importer.Directory)

	fmt.Println("Context:")
	fmt.Printf(" - MachineID: %s\n", header.GetContext("MachineID"))
	fmt.Printf(" - Hostname: %s\n", header.GetContext("Hostname"))
	fmt.Printf(" - Username: %s\n", header.GetContext("Username"))
	fmt.Printf(" - OperatingSystem: %s\n", header.GetContext("OperatingSystem"))
	fmt.Printf(" - Architecture: %s\n", header.GetContext("Architecture"))
	fmt.Printf(" - NumCPU: %s\n", header.GetContext("NumCPU"))
	fmt.Printf(" - GOMAXPROCS: %s\n", header.GetContext("GOMAXPROCS"))
	fmt.Printf(" - ProcessID: %s\n", header.GetContext("ProcessID"))
	fmt.Printf(" - Client: %s\n", header.GetContext("Client"))
	fmt.Printf(" - CommandLine: %s\n", header.GetContext("CommandLine"))

	fmt.Println("Summary:")
	fmt.Printf(" - Directories: %d\n", header.GetSource(0).Summary.Directory.Directories+header.GetSource(0).Summary.Below.Directories)
	fmt.Printf(" - Files: %d\n", header.GetSource(0).Summary.Directory.Files+header.GetSource(0).Summary.Below.Files)
	fmt.Printf(" - Symlinks: %d\n", header.GetSource(0).Summary.Directory.Symlinks+header.GetSource(0).Summary.Below.Symlinks)
	fmt.Printf(" - Devices: %d\n", header.GetSource(0).Summary.Directory.Devices+header.GetSource(0).Summary.Below.Devices)
	fmt.Printf(" - Pipes: %d\n", header.GetSource(0).Summary.Directory.Pipes+header.GetSource(0).Summary.Below.Pipes)
	fmt.Printf(" - Sockets: %d\n", header.GetSource(0).Summary.Directory.Sockets+header.GetSource(0).Summary.Below.Sockets)
	fmt.Printf(" - Setuid: %d\n", header.GetSource(0).Summary.Directory.Setuid+header.GetSource(0).Summary.Below.Setuid)
	fmt.Printf(" - Setgid: %d\n", header.GetSource(0).Summary.Directory.Setgid+header.GetSource(0).Summary.Below.Setgid)
	fmt.Printf(" - Sticky: %d\n", header.GetSource(0).Summary.Directory.Sticky+header.GetSource(0).Summary.Below.Sticky)

	fmt.Printf(" - Objects: %d\n", header.GetSource(0).Summary.Directory.Objects+header.GetSource(0).Summary.Below.Objects)
	fmt.Printf(" - Chunks: %d\n", header.GetSource(0).Summary.Directory.Chunks+header.GetSource(0).Summary.Below.Chunks)
	fmt.Printf(" - MinSize: %s (%d bytes)\n", humanize.IBytes(min(header.GetSource(0).Summary.Directory.MinSize, header.GetSource(0).Summary.Below.MinSize)), min(header.GetSource(0).Summary.Directory.MinSize, header.GetSource(0).Summary.Below.MinSize))
	fmt.Printf(" - MaxSize: %s (%d bytes)\n", humanize.IBytes(max(header.GetSource(0).Summary.Directory.MaxSize, header.GetSource(0).Summary.Below.MaxSize)), max(header.GetSource(0).Summary.Directory.MaxSize, header.GetSource(0).Summary.Below.MaxSize))
	fmt.Printf(" - Size: %s (%d bytes)\n", humanize.IBytes(header.GetSource(0).Summary.Directory.Size+header.GetSource(0).Summary.Below.Size), header.GetSource(0).Summary.Directory.Size+header.GetSource(0).Summary.Below.Size)
	fmt.Printf(" - MinModTime: %s\n", time.Unix(min(header.GetSource(0).Summary.Directory.MinModTime, header.GetSource(0).Summary.Below.MinModTime), 0))
	fmt.Printf(" - MaxModTime: %s\n", time.Unix(max(header.GetSource(0).Summary.Directory.MaxModTime, header.GetSource(0).Summary.Below.MaxModTime), 0))
	fmt.Printf(" - MinEntropy: %f\n", min(header.GetSource(0).Summary.Directory.MinEntropy, header.GetSource(0).Summary.Below.MinEntropy))
	fmt.Printf(" - MaxEntropy: %f\n", max(header.GetSource(0).Summary.Directory.MaxEntropy, header.GetSource(0).Summary.Below.MaxEntropy))
	fmt.Printf(" - HiEntropy: %d\n", header.GetSource(0).Summary.Directory.HiEntropy+header.GetSource(0).Summary.Below.HiEntropy)
	fmt.Printf(" - LoEntropy: %d\n", header.GetSource(0).Summary.Directory.LoEntropy+header.GetSource(0).Summary.Below.LoEntropy)
	fmt.Printf(" - MIMEAudio: %d\n", header.GetSource(0).Summary.Directory.MIMEAudio+header.GetSource(0).Summary.Below.MIMEAudio)
	fmt.Printf(" - MIMEVideo: %d\n", header.GetSource(0).Summary.Directory.MIMEVideo+header.GetSource(0).Summary.Below.MIMEVideo)
	fmt.Printf(" - MIMEImage: %d\n", header.GetSource(0).Summary.Directory.MIMEImage+header.GetSource(0).Summary.Below.MIMEImage)
	fmt.Printf(" - MIMEText: %d\n", header.GetSource(0).Summary.Directory.MIMEText+header.GetSource(0).Summary.Below.MIMEText)
	fmt.Printf(" - MIMEApplication: %d\n", header.GetSource(0).Summary.Directory.MIMEApplication+header.GetSource(0).Summary.Below.MIMEApplication)
	fmt.Printf(" - MIMEOther: %d\n", header.GetSource(0).Summary.Directory.MIMEOther+header.GetSource(0).Summary.Below.MIMEOther)

	fmt.Printf(" - Errors: %d\n", header.GetSource(0).Summary.Directory.Errors+header.GetSource(0).Summary.Below.Errors)
	return 0, nil
}
