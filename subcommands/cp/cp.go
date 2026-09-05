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

package cp

import (
	"flag"
	"fmt"
	"maps"
	"os"
	"path"
	"strings"
	"sync/atomic"
	"time"

	"github.com/PlakarKorp/kloset/connectors"
	"github.com/PlakarKorp/kloset/connectors/exporter"
	"github.com/PlakarKorp/kloset/connectors/importer"
	"github.com/PlakarKorp/kloset/location"
	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
)

type Cp struct {
	subcommands.SubcommandBase

	Src string
	Dst string

	Quiet bool
}

func init() {
	subcommands.Register(func() subcommands.Subcommand { return &Cp{} }, subcommands.BeforeRepositoryOpen, "cp")
}

func (cmd *Cp) Parse(ctx *appcontext.AppContext, args []string) error {
	flags := flag.NewFlagSet("cp", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [OPTIONS] SOURCE DESTINATION\n", flags.Name())
		flags.PrintDefaults()
	}
	flags.BoolVar(&cmd.Quiet, "quiet", false, "do not list the copied files")
	flags.Parse(args)

	if flags.NArg() != 2 {
		return fmt.Errorf("needs a source and a destination")
	}

	cmd.Src, cmd.Dst = flags.Arg(0), flags.Arg(1)
	return nil
}

// resolve turns a command line argument into a connector configuration.
// A leading "@" refers to a source or a destination from the
// configuration, anything else is used as a location as-is.
func resolve(arg string, lookup func(string) (map[string]string, bool)) (map[string]string, error) {
	if !strings.HasPrefix(arg, "@") {
		return map[string]string{"location": arg}, nil
	}

	kv, ok := lookup(arg[1:])
	if !ok {
		return nil, fmt.Errorf("could not resolve %s", arg)
	}
	if _, found := kv["location"]; !found {
		return nil, fmt.Errorf("could not resolve location for %s", arg)
	}

	conf := make(map[string]string)
	maps.Copy(conf, kv)
	return conf, nil
}

// parentOf returns the directory holding pathname, or "" for the root
// and for the relative paths an importer has no parent to report for.
func parentOf(pathname string) string {
	if pathname == "/" || !strings.HasPrefix(pathname, "/") {
		return ""
	}

	idx := strings.LastIndex(pathname, "/")
	if idx <= 0 {
		return "/"
	}
	return pathname[:idx]
}

// rebase strips the directory holding root from pathname, so that
// copying /private/etc lands in etc/ under the destination instead of
// recreating the whole path down to it.  It's the same chroot the
// restore of a snapshot simulates, and what cp(1) does with a
// directory.
//
// Paths outside of root are not part of the copy: the fs importer
// reports the directories leading to its root, and recreating those
// would litter the destination with empty ones.
func rebase(pathname, root string) (string, bool) {
	if pathname == root {
		return "/" + path.Base(root), true
	}
	if !strings.HasPrefix(pathname, strings.TrimSuffix(root, "/")+"/") {
		return "", false
	}

	parent := parentOf(root)
	if parent == "" || parent == "/" {
		return pathname, true
	}
	return pathname[len(parent):], true
}

// dirRecord makes up the directory at pathname, for the ones an
// importer doesn't report.  It mirrors what the vfs builder does when
// it relinks a tree, down to the mode and the zeroed mtime.
func dirRecord(pathname string) *connectors.Record {
	return &connectors.Record{
		Pathname: pathname,
		FileInfo: objects.FileInfo{
			Lname:    path.Base(pathname),
			Lmode:    os.ModeDir | 0750,
			LmodTime: time.Unix(0, 0).UTC(),
		},
	}
}

// feed rebases the records under root and hands them over to send,
// making sure a directory always comes before the entries it holds.
//
// The fs importer walks the tree with a pool of workers, so a child can
// come out before its parent, and an importer only reports what it was
// asked for: the directories above its root are not part of the copy
// once the paths are rebased, but the ones in between still have to be
// there.  A backup doesn't have to care because the vfs relinks the
// tree and makes up whatever the importer left out; cp goes straight
// from the importer to the exporter, so it makes them up itself.
//
// The made up directories go out before the entry that needed them,
// which settles the ordering too: by the time an entry is sent, every
// directory above it has been.  Exporters with no directories to speak
// of, like s3, are content to receive them and ignore them.
func feed(records <-chan *connectors.Record, root string, send func(*connectors.Record), fail func(*connectors.Record)) {
	sent := make(map[string]bool)

	// sendDirs walks up to the first directory already sent, then
	// sends what it skipped on the way back down.
	var sendDirs func(string)
	sendDirs = func(pathname string) {
		// "/" is the destination itself, it needs no record.
		if pathname == "" || pathname == "/" || sent[pathname] {
			return
		}
		sendDirs(parentOf(pathname))

		sent[pathname] = true
		send(dirRecord(pathname))
	}

	for record := range records {
		if record.Err != nil {
			fail(record)
			continue
		}

		pathname, ok := rebase(record.Pathname, root)
		if !ok {
			record.Close()
			continue
		}

		// The importer reads the file back by pathname, so rebase a
		// copy of the record and leave the one it handed us alone.
		out := *record
		out.Pathname = pathname

		sendDirs(parentOf(out.Pathname))

		// A directory the importer did report supersedes anything we
		// would have made up for it: it carries the real mode and
		// mtime, and the exporter takes the second mkdir(2) as a
		// no-op.
		if out.FileInfo.Lmode.IsDir() {
			sent[out.Pathname] = true
		}
		send(&out)
	}
}

func (cmd *Cp) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	src, err := resolve(cmd.Src, ctx.Config.GetSource)
	if err != nil {
		return 1, err
	}

	dst, err := resolve(cmd.Dst, ctx.Config.GetDestination)
	if err != nil {
		return 1, err
	}

	imp, err := importer.NewImporter(ctx.GetInner(), ctx.ImporterOpts(), src)
	if err != nil {
		return 1, err
	}
	defer imp.Close(ctx)

	exp, err := exporter.NewExporter(ctx.GetInner(), ctx.ExporterOpts(), dst)
	if err != nil {
		return 1, err
	}
	defer exp.Close(ctx)

	var (
		erec = make(chan *connectors.Record)
		eres = make(chan *connectors.Result)
		done = make(chan error, 1)
	)

	go func() {
		done <- exp.Export(ctx, erec, eres)
	}()

	// number of records the importer failed to produce, or that the
	// exporter failed to consume; bumped from the feeding loop and
	// from the acknowledgement one below.
	var failed atomic.Uint64

	err = drive(ctx, imp, func(records <-chan *connectors.Record, results chan<- *connectors.Result) {
		// Drain the exporter acknowledgements concurrently with the
		// feeding loop below: Export() may block on eres until we
		// read from it. ackdone lets us wait for this goroutine
		// before returning, so that all the acks are accounted for.
		ackdone := make(chan struct{})
		go func() {
			defer close(ackdone)
			for res := range eres {
				if res.Err != nil {
					failed.Add(1)
					ctx.GetLogger().Error("%s: %s", res.Record.Pathname, res.Err)
				} else if !cmd.Quiet {
					fmt.Fprintln(ctx.Stdout, res.Record.Pathname)
				}
				res.Record.Close()
				if results != nil {
					results <- res
				}
			}
		}()

		send := func(record *connectors.Record) {
			select {
			case erec <- record:
			case <-ctx.Done():
				record.Close()
			}
		}

		feed(records, imp.Root(), send, func(record *connectors.Record) {
			// The importer reports per-record failures inline; such
			// records carry no payload, so don't hand them over to
			// the exporter.
			failed.Add(1)
			ctx.GetLogger().Error("%s: %s", record.Pathname, record.Err)
			record.Close()
			if results != nil {
				results <- record.Error(record.Err)
			}
		})

		// Signals the exporter that no more records are coming; it
		// then closes eres, which terminates the drain above.
		close(erec)
		<-ackdone
	})
	if err != nil {
		return 1, err
	}

	if err := <-done; err != nil {
		return 1, err
	}

	if n := failed.Load(); n != 0 {
		return 1, fmt.Errorf("failed to copy %d entries", n)
	}
	return 0, nil
}

// drive runs the importer and the given function concurrently, wiring
// the record and (when the importer asks for them) the result channels
// between the two.
func drive(ctx *appcontext.AppContext, imp importer.Importer, fn func(<-chan *connectors.Record, chan<- *connectors.Result)) error {
	var (
		size    = ctx.MaxConcurrency
		records = make(chan *connectors.Record, size)
		retch   = make(chan struct{})
	)

	var results chan *connectors.Result
	if (imp.Flags() & location.FLAG_NEEDACK) != 0 {
		results = make(chan *connectors.Result, size)
	}

	go func() {
		defer close(retch)
		fn(records, results)
		if results != nil {
			close(results)
		}
		// Import() may still be trying to hand us records if it
		// bailed out early; drain them so that it doesn't block.
		for record := range records {
			record.Close()
		}
	}()

	err := imp.Import(ctx, records, results)
	<-retch
	return err
}
