package main

import (
	"crypto/sha256"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/poolpOrg/plakar/store"
)

func cmd_verify(pstore store.Store, args []string) {
	if len(args) == 0 {
		log.Fatalf("%s: need at least one snapshot ID to check", flag.CommandLine.Name())
	}

	snapshots := make([]string, 0)
	for id, _ := range pstore.Snapshots() {
		snapshots = append(snapshots, id)
	}

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
		unlistedChunk := make([]string, 0)
		missingChunks := make([]string, 0)
		corruptedChunks := make([]string, 0)
		unlistedObject := make([]string, 0)
		missingObjects := make([]string, 0)
		corruptedObjects := make([]string, 0)
		unlistedFile := make([]string, 0)

		prefix, pattern := parseSnapshotID(args[i])
		res := findSnapshotByPrefix(snapshots, prefix)
		snapshot := pstore.Snapshot(res[0])

		if pattern != "" {
			checksum, ok := snapshot.Sums[pattern]
			if !ok {
				unlistedFile = append(unlistedFile, pattern)
				continue
			}
			object, ok := snapshot.Objects[checksum]
			if !ok {
				unlistedObject = append(unlistedObject, checksum)
				continue
			}

			objectHash := sha256.New()
			for _, chunk := range object.Chunks {
				data, err := snapshot.ChunkGet(chunk.Checksum)
				if err != nil {
					missingChunks = append(missingChunks, chunk.Checksum)
					continue
				}
				objectHash.Write(data)
			}
			if fmt.Sprintf("%032x", objectHash.Sum(nil)) != checksum {
				corruptedObjects = append(corruptedObjects, checksum)
				continue
			}

		} else {

			cCount := 0
			for _, chunk := range snapshot.Chunks {
				data, err := snapshot.ChunkGet(chunk.Checksum)
				if err != nil {
					missingChunks = append(missingChunks, chunk.Checksum)
					continue
				}
				chunkHash := sha256.New()
				chunkHash.Write(data)
				if fmt.Sprintf("%032x", chunkHash.Sum(nil)) != chunk.Checksum {
					corruptedChunks = append(corruptedChunks, chunk.Checksum)
					continue
				}
				cCount += 1

				if !quiet {
					fmt.Fprintf(os.Stdout, "\r%s: chunks: %2d%%",
						snapshot.Uuid,
						(cCount*100)/len(snapshot.Chunks))
				}
			}

			oCount := 0
			for checksum, _ := range snapshot.Objects {
				object, err := snapshot.ObjectGet(checksum)
				if err != nil {
					missingObjects = append(missingObjects, checksum)
					continue
				}
				objectHash := sha256.New()

				for _, chunk := range object.Chunks {
					_, ok := snapshot.Chunks[chunk.Checksum]
					if !ok {
						unlistedChunk = append(unlistedChunk, chunk.Checksum)
						continue
					}

					data, err := snapshot.ChunkGet(chunk.Checksum)
					if err != nil {
						missingChunks = append(missingChunks, chunk.Checksum)
						continue
					}
					objectHash.Write(data)
				}
				if fmt.Sprintf("%032x", objectHash.Sum(nil)) != checksum {
					corruptedObjects = append(corruptedObjects, checksum)
					continue
				}

				oCount += 1
				if !quiet {
					fmt.Fprintf(os.Stdout, "\r%s: objects: %2d%%",
						snapshot.Uuid,
						(oCount*100)/len(snapshot.Objects))
				}
			}

			fCount := 0
			for file, _ := range snapshot.Files {
				checksum, ok := snapshot.Sums[file]
				if !ok {
					unlistedFile = append(unlistedFile, file)
					continue
				}
				_, ok = snapshot.Objects[checksum]
				if !ok {
					unlistedObject = append(unlistedObject, checksum)
					continue
				}

				fCount += 1
				if !quiet {
					fmt.Fprintf(os.Stdout, "\r%s: files: %2d%%  ",
						snapshot.Uuid,
						(fCount*100)/len(snapshot.Files))
				}
			}
		}

		if !quiet {
			errors := 0
			errors += len(missingChunks)
			errors += len(corruptedChunks)
			errors += len(missingObjects)
			errors += len(corruptedObjects)
			errors += len(unlistedChunk)

			key := snapshot.Uuid
			if pattern != "" {
				key = fmt.Sprintf("%s:%s", snapshot.Uuid, pattern)
			}

			if errors == 0 {
				fmt.Fprintf(os.Stdout, "\r%s: OK         \n", key)
			} else {
				fmt.Fprintf(os.Stdout, "\r%s: KO         \n", key)
			}
		}
	}
}