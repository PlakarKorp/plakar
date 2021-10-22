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
	"crypto/sha256"
	"flag"
	"fmt"
	"log"

	"github.com/poolpOrg/plakar/logger"
	"github.com/poolpOrg/plakar/snapshot"
)

func cmd_check(ctx Plakar, args []string) int {
	flags := flag.NewFlagSet("check", flag.ExitOnError)
	flags.Parse(args)

	if len(flags.Args()) == 0 {
		logger.Error("%s: at least one parameter is required", flags.Name())
		return 1
	}

	snapshots := getSnapshotsList(ctx)
	checkSnapshotsArgs(snapshots)

	for i := 0; i < len(args); i++ {
		prefix, pattern := parseSnapshotID(args[i])
		res := findSnapshotByPrefix(snapshots, prefix)
		snap, err := snapshot.Load(ctx.Store(), res[0])
		if err != nil {
			fmt.Println(err)
			log.Fatalf("%s: could not open snapshot %s", flag.CommandLine.Name(), res[0])
		}

		snapshotOk := false
		if pattern != "" {
			checksum, ok := snap.Pathnames[pattern]
			if !ok {
				logger.Warn("%s: unlisted file %s", snap.Uuid, pattern)
				snapshotOk = false
				continue
			}
			object, ok := snap.Objects[checksum]
			if !ok {
				logger.Warn("%s: unlisted object %s", snap.Uuid, checksum)
				snapshotOk = false
				continue
			}

			objectHash := sha256.New()
			for _, chunk := range object.Chunks {
				data, err := snap.GetChunk(chunk.Checksum)
				if err != nil {
					logger.Warn("%s: missing chunk %s", snap.Uuid, chunk.Checksum)
					snapshotOk = false
					continue
				}
				objectHash.Write(data)
			}
			if fmt.Sprintf("%032x", objectHash.Sum(nil)) != checksum {
				logger.Warn("%s: corrupted object %s", snap.Uuid, checksum)
				snapshotOk = false
				continue
			}

		} else {

			for _, chunk := range snap.Chunks {

				data, err := snap.GetChunk(chunk.Checksum)
				if err != nil {
					logger.Warn("%s: missing chunk %s", snap.Uuid, chunk.Checksum)
					snapshotOk = false
					continue
				}
				chunkHash := sha256.New()
				chunkHash.Write(data)
				if fmt.Sprintf("%032x", chunkHash.Sum(nil)) != chunk.Checksum {
					logger.Warn("%s: corrupted chunk %s", snap.Uuid, chunk.Checksum)
					snapshotOk = false
					continue
				}

			}

			for checksum := range snap.Objects {
				object, err := snap.GetObject(checksum)
				if err != nil {
					logger.Warn("%s: missing object %s", snap.Uuid, checksum)
					snapshotOk = false
					continue
				}
				objectHash := sha256.New()

				for _, chunk := range object.Chunks {
					_, ok := snap.Chunks[chunk.Checksum]
					if !ok {
						logger.Warn("%s: unlisted chunk %s", snap.Uuid, chunk.Checksum)
						snapshotOk = false
						continue
					}

					data, err := snap.GetChunk(chunk.Checksum)
					if err != nil {
						logger.Warn("%s: missing chunk %s", snap.Uuid, chunk.Checksum)
						snapshotOk = false
						continue
					}
					objectHash.Write(data)
				}
				if fmt.Sprintf("%032x", objectHash.Sum(nil)) != checksum {
					logger.Warn("%s: corrupted object %s", snap.Uuid, checksum)
					snapshotOk = false
					continue
				}
			}

			for file := range snap.Files {
				checksum, ok := snap.Pathnames[file]
				if !ok {
					logger.Warn("%s: unlisted file %s", snap.Uuid, file)
					snapshotOk = false
					continue
				}
				_, ok = snap.Objects[checksum]
				if !ok {
					logger.Warn("%s: unlisted object %s", snap.Uuid, checksum)
					snapshotOk = false
					continue
				}
			}
		}

		key := snap.Uuid
		if pattern != "" {
			key = fmt.Sprintf("%s:%s", snap.Uuid, pattern)
		}
		_ = key

		if snapshotOk {
			logger.Info("%s: OK", snap.Uuid)
		} else {
			logger.Error("%s: KO", snap.Uuid)
		}
	}
	return 0
}
