package info

import (
	"fmt"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/dustin/go-humanize"
)

func (cmd *Info) executeRepository(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {

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
		fmt.Println(" - SubkeyAlgorithm:", repo.Configuration().Encryption.SubKeyAlgorithm)
		fmt.Println(" - DataAlgorithm:", repo.Configuration().Encryption.DataAlgorithm)
		fmt.Println(" - ChunkSize:", repo.Configuration().Encryption.ChunkSize)
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

	nSnapshots, logicalSize, err := snapshot.LogicalSize(repo)
	if err != nil {
		return 1, fmt.Errorf("unable to calculate logical size: %w", err)
	}

	fmt.Println("Snapshots:", nSnapshots)

	storageSize, err := repo.Store().Size(ctx)
	if err != nil {
		return 1, fmt.Errorf("unable to compute storage size: %w", err)
	}

	fmt.Printf("Storage size: %s (%d bytes)\n", humanize.IBytes(uint64(storageSize)), uint64(storageSize))
	fmt.Printf("Logical size: %s (%d bytes)\n", humanize.IBytes(uint64(logicalSize)), logicalSize)

	return 0, nil
}
