/*
 * Copyright (c) 2021 Gilles Chehade <gilles@poolp.org>
 *
 * Permission to use, copy, modify, and distribute this software for any
 * purpose with or without fee is hereby granted, provided that the above
 * copyright notice and this permission notice appear in all copies.
 *
 * THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
 * WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
 * MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
 * ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
 * WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
 * ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
 * OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
 */

package clone

import (
	"bytes"
	"flag"
	"fmt"
	"hash"
	"io"
	"os"

	"github.com/PlakarKorp/kloset/hashing"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/resources"
	"github.com/PlakarKorp/kloset/storage"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/google/uuid"
)

func init() {
	subcommands.Register(func() subcommands.Subcommand { return &Clone{} }, 0, "clone")
}

func (cmd *Clone) Parse(ctx *appcontext.AppContext, args []string) error {
	flags := flag.NewFlagSet("clone", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s to /path/to/repository\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       %s to s3://bucket/path\n", flags.Name())
		flags.PrintDefaults()
	}

	flags.Parse(args)

	if flags.NArg() != 2 || flags.Arg(0) != "to" {
		return fmt.Errorf("usage: %s to <repository>. See '%s -h' or 'help %s'", flags.Name(), flags.Name(), flags.Name())
	}

	cmd.RepositorySecret = ctx.GetSecret()
	cmd.Dest = flags.Arg(1)

	return nil
}

type Clone struct {
	subcommands.SubcommandBase

	Dest string
}

func (cmd *Clone) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	configuration := repo.Configuration()

	configuration.RepositoryID = uuid.New()

	serializedConfig, err := configuration.ToBytes()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to decode storage configuration: %s\n", err)
		return 1, err
	}

	var hasher hash.Hash
	if configuration.Encryption != nil {
		hasher = hashing.GetMACHasher(storage.DEFAULT_HASHING_ALGORITHM, ctx.GetSecret())
	} else {
		hasher = hashing.GetHasher(storage.DEFAULT_HASHING_ALGORITHM)
	}

	wrappedSerializedConfigRd, err := storage.Serialize(hasher,
		resources.RT_CONFIG, repo.Configuration().Version, bytes.NewReader(serializedConfig))
	if err != nil {
		return 1, err
	}
	wrappedSerializedConfig, err := io.ReadAll(wrappedSerializedConfigRd)
	if err != nil {
		return 1, err
	}

	storeConfig, err := ctx.Config.GetRepository(cmd.Dest)
	if err != nil {
		return 1, err
	}

	_, err = storage.Create(ctx.GetInner(), storeConfig, wrappedSerializedConfig)
	if err != nil {
		return 1, fmt.Errorf("could not clone store: %w", err)
	}

	return 0, nil
}
