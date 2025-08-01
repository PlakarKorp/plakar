package info

import (
	"fmt"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/dustin/go-humanize"
)

func (cmd *Info) executeRepository(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {

	fmt.Fprintln(ctx.Stdout, "Version:", repo.Configuration().Version)
	fmt.Fprintln(ctx.Stdout, "Timestamp:", repo.Configuration().Timestamp)
	fmt.Fprintln(ctx.Stdout, "RepositoryID:", repo.Configuration().RepositoryID)

	fmt.Fprintln(ctx.Stdout, "Packfile:")
	fmt.Fprintf(ctx.Stdout, " - MaxSize: %s (%d bytes)\n",
		humanize.IBytes(uint64(repo.Configuration().Packfile.MaxSize)),
		repo.Configuration().Packfile.MaxSize)

	fmt.Fprintln(ctx.Stdout, "Chunking:")
	fmt.Fprintln(ctx.Stdout, " - Algorithm:", repo.Configuration().Chunking.Algorithm)
	fmt.Fprintf(ctx.Stdout, " - MinSize: %s (%d bytes)\n",
		humanize.IBytes(uint64(repo.Configuration().Chunking.MinSize)), repo.Configuration().Chunking.MinSize)
	fmt.Fprintf(ctx.Stdout, " - NormalSize: %s (%d bytes)\n",
		humanize.IBytes(uint64(repo.Configuration().Chunking.NormalSize)), repo.Configuration().Chunking.NormalSize)
	fmt.Fprintf(ctx.Stdout, " - MaxSize: %s (%d bytes)\n",
		humanize.IBytes(uint64(repo.Configuration().Chunking.MaxSize)), repo.Configuration().Chunking.MaxSize)

	fmt.Fprintln(ctx.Stdout, "Hashing:")
	fmt.Fprintln(ctx.Stdout, " - Algorithm:", repo.Configuration().Hashing.Algorithm)
	fmt.Fprintln(ctx.Stdout, " - Bits:", repo.Configuration().Hashing.Bits)

	if repo.Configuration().Compression != nil {
		fmt.Fprintln(ctx.Stdout, "Compression:")
		fmt.Fprintln(ctx.Stdout, " - Algorithm:", repo.Configuration().Compression.Algorithm)
		fmt.Fprintln(ctx.Stdout, " - Level:", repo.Configuration().Compression.Level)
	}

	if repo.Configuration().Encryption != nil {
		fmt.Fprintln(ctx.Stdout, "Encryption:")
		fmt.Fprintln(ctx.Stdout, " - SubkeyAlgorithm:", repo.Configuration().Encryption.SubKeyAlgorithm)
		fmt.Fprintln(ctx.Stdout, " - DataAlgorithm:", repo.Configuration().Encryption.DataAlgorithm)
		fmt.Fprintln(ctx.Stdout, " - ChunkSize:", repo.Configuration().Encryption.ChunkSize)
		fmt.Fprintf(ctx.Stdout, " - Canary: %x\n", repo.Configuration().Encryption.Canary)
		fmt.Fprintln(ctx.Stdout, " - KDF:", repo.Configuration().Encryption.KDFParams.KDF)
		fmt.Fprintf(ctx.Stdout, "   - Salt: %x\n", repo.Configuration().Encryption.KDFParams.Salt)
		switch repo.Configuration().Encryption.KDFParams.KDF {
		case "ARGON2ID":
			fmt.Fprintf(ctx.Stdout, "   - SaltSize: %d\n", repo.Configuration().Encryption.KDFParams.Argon2idParams.SaltSize)
			fmt.Fprintf(ctx.Stdout, "   - KeyLen: %d\n", repo.Configuration().Encryption.KDFParams.Argon2idParams.KeyLen)
			fmt.Fprintf(ctx.Stdout, "   - Time: %d\n", repo.Configuration().Encryption.KDFParams.Argon2idParams.Time)
			fmt.Fprintf(ctx.Stdout, "   - Memory: %d\n", repo.Configuration().Encryption.KDFParams.Argon2idParams.Memory)
			fmt.Fprintf(ctx.Stdout, "   - Thread: %d\n", repo.Configuration().Encryption.KDFParams.Argon2idParams.Threads)
		case "SCRYPT":
			fmt.Fprintf(ctx.Stdout, "   - SaltSize: %d\n", repo.Configuration().Encryption.KDFParams.ScryptParams.SaltSize)
			fmt.Fprintf(ctx.Stdout, "   - KeyLen: %d\n", repo.Configuration().Encryption.KDFParams.ScryptParams.KeyLen)
			fmt.Fprintf(ctx.Stdout, "   - N: %d\n", repo.Configuration().Encryption.KDFParams.ScryptParams.N)
			fmt.Fprintf(ctx.Stdout, "   - R: %d\n", repo.Configuration().Encryption.KDFParams.ScryptParams.R)
			fmt.Fprintf(ctx.Stdout, "   - P: %d\n", repo.Configuration().Encryption.KDFParams.ScryptParams.P)
		case "PBKDF2":
			fmt.Fprintf(ctx.Stdout, "   - SaltSize: %d\n", repo.Configuration().Encryption.KDFParams.Pbkdf2Params.SaltSize)
			fmt.Fprintf(ctx.Stdout, "   - KeyLen: %d\n", repo.Configuration().Encryption.KDFParams.Pbkdf2Params.KeyLen)
			fmt.Fprintf(ctx.Stdout, "   - Iterations: %d\n", repo.Configuration().Encryption.KDFParams.Pbkdf2Params.Iterations)
			fmt.Fprintf(ctx.Stdout, "   - Hashing: %s\n", repo.Configuration().Encryption.KDFParams.Pbkdf2Params.Hashing)
		default:
			fmt.Fprintf(ctx.Stdout, "   - Unsupported KDF: %s\n", repo.Configuration().Encryption.KDFParams.KDF)
		}
	}

	nSnapshots, logicalSize, err := snapshot.LogicalSize(repo)
	if err != nil {
		return 1, fmt.Errorf("unable to calculate logical size: %w", err)
	}

	fmt.Fprintln(ctx.Stdout, "Snapshots:", nSnapshots)

	storageSize, err := repo.Store().Size(ctx)
	if err != nil {
		return 1, fmt.Errorf("unable to compute storage size: %w", err)
	}

	fmt.Fprintf(ctx.Stdout, "Storage size: %s (%d bytes)\n", humanize.IBytes(uint64(storageSize)), uint64(storageSize))
	fmt.Fprintf(ctx.Stdout, "Logical size: %s (%d bytes)\n", humanize.IBytes(uint64(logicalSize)), logicalSize)

	return 0, nil
}
