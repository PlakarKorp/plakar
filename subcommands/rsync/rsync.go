/*
 * Copyright (c) 2026 Omar Polo <op@omarpolo.com>
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

package rsync

import (
	"flag"
	"fmt"

	"github.com/PlakarKorp/kloset/connectors"
	"github.com/PlakarKorp/kloset/connectors/exporter"
	"github.com/PlakarKorp/kloset/connectors/importer"
	"github.com/PlakarKorp/kloset/location"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
)

type Rsync struct {
	subcommands.SubcommandBase

	Src string
	Dst string
}

func init() {
	subcommands.Register(func() subcommands.Subcommand { return &Rsync{} }, subcommands.BeforeRepositoryOpen, "rsync")
}

func (cmd *Rsync) Parse(ctx *appcontext.AppContext, args []string) error {
	flags := flag.NewFlagSet("rsync", flag.ExitOnError)
	flags.Parse(args)

	if flags.NArg() != 2 {
		return fmt.Errorf("Too many arguments")
	}

	cmd.Src, cmd.Dst = flags.Arg(0), flags.Arg(1)
	return nil
}

func (cmd *Rsync) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	srcConf, ok := ctx.Config.GetSource(cmd.Src[1:])
	if !ok {
		srcConf = map[string]string{
			"location": cmd.Src,
		}
	}

	imp, err := importer.NewImporter(ctx.GetInner(), ctx.ImporterOpts(),
		srcConf)
	if err != nil {
		return 1, err
	}

	dstConf, ok := ctx.Config.GetSource(cmd.Dst[1:])
	if !ok {
		dstConf = map[string]string{
			"location": cmd.Dst,
		}
	}

	exp, err := exporter.NewExporter(ctx.GetInner(), ctx.ExporterOpts(),
		dstConf)
	if err != nil {
		return 1, err
	}

	var (
		erec = make(chan *connectors.Record)
		eres = make(chan *connectors.Result)
		done = make(chan error)
	)

	go func() {
		done <- exp.Export(ctx, erec, eres)
		close(done)
	}()

	err = progress(ctx, imp, func(records <-chan *connectors.Record, results chan<- *connectors.Result) {
		go func() {
			for res := range eres {
				fmt.Println("<-", res.Record.Pathname)
				res.Record.Close()
				if results != nil {
					results <- res
				}
			}
		}()

		for record := range records {
			fmt.Println("->", record.Pathname)
			erec <- record
		}
	})
	if err != nil {
		return 1, err
	}

	err = <-done
	if err != nil {
		return 1, err
	}
	return 0, nil
}

func progress(ctx *appcontext.AppContext, imp importer.Importer, fn func(<-chan *connectors.Record, chan<- *connectors.Result)) error {
	var (
		size    = ctx.MaxConcurrency
		records = make(chan *connectors.Record, size)
		retch   = make(chan struct{}, 1)
	)

	var results chan *connectors.Result
	if (imp.Flags() & location.FLAG_NEEDACK) != 0 {
		results = make(chan *connectors.Result, size)
	}

	go func() {
		fn(records, results)
		if results != nil {
			close(results)
		}
		close(retch)
	}()

	err := imp.Import(ctx, records, results)
	<-retch
	return err
}
