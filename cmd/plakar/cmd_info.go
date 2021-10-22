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

package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/dustin/go-humanize"
	"github.com/poolpOrg/plakar/snapshot"
)

func cmd_info(ctx Plakar, args []string) int {
	if len(args) == 0 {
		log.Fatalf("%s: need at least one snapshot ID to pull", flag.CommandLine.Name())
	}

	snapshots := getSnapshotsList(ctx)

	for i := 0; i < len(args); i++ {
		prefix, _ := parseSnapshotID(args[i])
		res := findSnapshotByPrefix(snapshots, prefix)
		if len(res) == 0 {
			log.Fatalf("%s: no snapshot has prefix: %s", flag.CommandLine.Name(), prefix)
		} else if len(res) > 1 {
			log.Fatalf("%s: snapshot ID is ambigous: %s (matches %d snapshots)", flag.CommandLine.Name(), prefix, len(res))
		}
	}

	for i := 0; i < len(args); i++ {
		prefix, _ := parseSnapshotID(args[i])
		res := findSnapshotByPrefix(snapshots, prefix)
		snap, err := snapshot.Load(ctx.Store(), res[0])
		if err != nil {
			log.Fatalf("%s: could not open snapshot %s", flag.CommandLine.Name(), res[0])
		}
		fmt.Printf("Uuid: %s\n", snap.Uuid)
		fmt.Printf("CreationTime: %s\n", snap.CreationTime)
		fmt.Printf("Version: %s\n", snap.Version)
		fmt.Printf("Hostname: %s\n", snap.Hostname)
		fmt.Printf("Username: %s\n", snap.Username)
		fmt.Printf("Directories: %d\n", len(snap.Directories))
		fmt.Printf("Files: %d\n", len(snap.Files))
		fmt.Printf("NonRegular: %d\n", len(snap.NonRegular))
		fmt.Printf("Sums: %d\n", len(snap.Pathnames))
		fmt.Printf("Objects: %d\n", len(snap.Objects))
		fmt.Printf("Chunks: %d\n", len(snap.Chunks))
		fmt.Printf("Size: %s (%d bytes)\n", humanize.Bytes(snap.Size), snap.Size)
	}

	return 0
}
