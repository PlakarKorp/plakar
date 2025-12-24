package diag

import (
	"flag"
	"fmt"

	"github.com/PlakarKorp/kloset/locate"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/dustin/go-humanize"
)

type DiagRepository struct {
	subcommands.SubcommandBase

	RepositoryLocation string
}

func (cmd *DiagRepository) Parse(ctx *appcontext.AppContext, args []string) error {
	// Since this is the default action, we plug the general USAGE here.
	flags := flag.NewFlagSet("diag", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       %s snapshot SNAPSHOT\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       %s errors SNAPSHOT\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       %s state [STATE]...\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       %s search snapshot[:path] mime\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       %s packfile [PACKFILE]...\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       %s object [OBJECT]\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       %s vfs SNAPSHOT[:PATH]\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       %s xattr SNAPSHOT[:PATH]\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       %s contenttype SNAPSHOT[:PATH]\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       %s locks\n", flags.Name())
	}
	flags.Parse(args)

	cmd.RepositorySecret = ctx.GetSecret()

	return nil
}

func (cmd *DiagRepository) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {

	fmt.Println("Version:", repo.Configuration().Version)
	fmt.Println("Timestamp:", repo.Configuration().Timestamp)
	fmt.Println("RepositoryID:", repo.Configuration().RepositoryID)

	fmt.Println("Packfile:")
	fmt.Printf(" - MaxSize: %s (%d bytes)\n",
		humanize.IBytes(uint64(repo.Configuration().Packfile.MaxSize)),
		repo.Configuration().Packfile.MaxSize)

	fmt.Println("Chunking:")
	fmt.Println(" - Algorithm:", repo.Configuration().Chunking.Algorithm)
	fmt.Printf(" - MinSize: %s (%d bytes)\n",
		humanize.IBytes(uint64(repo.Configuration().Chunking.MinSize)), repo.Configuration().Chunking.MinSize)
	fmt.Printf(" - NormalSize: %s (%d bytes)\n",
		humanize.IBytes(uint64(repo.Configuration().Chunking.NormalSize)), repo.Configuration().Chunking.NormalSize)
	fmt.Printf(" - MaxSize: %s (%d bytes)\n",
		humanize.IBytes(uint64(repo.Configuration().Chunking.MaxSize)), repo.Configuration().Chunking.MaxSize)

	fmt.Println("Hashing:")
	fmt.Println(" - Algorithm:", repo.Configuration().Hashing.Algorithm)
	fmt.Println(" - Bits:", repo.Configuration().Hashing.Bits)

	if repo.Configuration().Compression != nil {
		fmt.Println("Compression:")
		fmt.Println(" - Algorithm:", repo.Configuration().Compression.Algorithm)
		fmt.Println(" - Level:", repo.Configuration().Compression.Level)
	}

	if repo.Configuration().Encryption != nil {
		fmt.Println("Encryption:")
		fmt.Println(" - Data Algorithm:", repo.Configuration().Encryption.DataAlgorithm)
		fmt.Println(" - Subkey Algorithm:", repo.Configuration().Encryption.SubKeyAlgorithm)
		fmt.Printf(" - Canary: %x\n", repo.Configuration().Encryption.Canary)
		fmt.Println(" - KDF:", repo.Configuration().Encryption.KDFParams.KDF)
		fmt.Printf("   - Salt: %x\n", repo.Configuration().Encryption.KDFParams.Salt)
		switch repo.Configuration().Encryption.KDFParams.KDF {
		case "ARGON2ID":
			fmt.Printf("   - SaltSize: %d\n", repo.Configuration().Encryption.KDFParams.Argon2idParams.SaltSize)
			fmt.Printf("   - KeyLen: %d\n", repo.Configuration().Encryption.KDFParams.Argon2idParams.KeyLen)
			fmt.Printf("   - Time: %d\n", repo.Configuration().Encryption.KDFParams.Argon2idParams.Time)
			fmt.Printf("   - Memory: %d\n", repo.Configuration().Encryption.KDFParams.Argon2idParams.Memory)
			fmt.Printf("   - Thread: %d\n", repo.Configuration().Encryption.KDFParams.Argon2idParams.Threads)
		case "SCRYPT":
			fmt.Printf("   - SaltSize: %d\n", repo.Configuration().Encryption.KDFParams.ScryptParams.SaltSize)
			fmt.Printf("   - KeyLen: %d\n", repo.Configuration().Encryption.KDFParams.ScryptParams.KeyLen)
			fmt.Printf("   - N: %d\n", repo.Configuration().Encryption.KDFParams.ScryptParams.N)
			fmt.Printf("   - R: %d\n", repo.Configuration().Encryption.KDFParams.ScryptParams.R)
			fmt.Printf("   - P: %d\n", repo.Configuration().Encryption.KDFParams.ScryptParams.P)
		case "PBKDF2":
			fmt.Printf("   - SaltSize: %d\n", repo.Configuration().Encryption.KDFParams.Pbkdf2Params.SaltSize)
			fmt.Printf("   - KeyLen: %d\n", repo.Configuration().Encryption.KDFParams.Pbkdf2Params.KeyLen)
			fmt.Printf("   - Iterations: %d\n", repo.Configuration().Encryption.KDFParams.Pbkdf2Params.Iterations)
			fmt.Printf("   - Hashing: %s\n", repo.Configuration().Encryption.KDFParams.Pbkdf2Params.Hashing)
		default:
			fmt.Printf("   - Unsupported KDF: %s\n", repo.Configuration().Encryption.KDFParams.KDF)
		}
	}

	snapshotIDs, err := locate.LocateSnapshotIDs(repo, nil)
	if err != nil {
		return 1, err
	}

	fmt.Println("Snapshots:", len(snapshotIDs))
	totalSize := uint64(0)
	for _, snapshotID := range snapshotIDs {
		snap, err := snapshot.Load(repo, snapshotID)
		if err != nil {
			return 1, err
		}
		totalSize += snap.Header.GetSource(0).Summary.Directory.Size + snap.Header.GetSource(0).Summary.Below.Size
		snap.Close()
	}
	fmt.Printf("Size: %s (%d bytes)\n", humanize.IBytes(totalSize), totalSize)

	return 0, nil
}
