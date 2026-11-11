/*
 * Copyright (c) 2026 Gilles Chehade <gilles@poolp.org>
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

package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/resources"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/kloset/snapshot/vfs"
)

// IntegrityFailure is one file whose data does not read back as it was
// written: a chunk gone from the store, or one whose MAC no longer matches.
type IntegrityFailure struct {
	Path  string `json:"path"`
	Error string `json:"error"`
}

// IntegrityReport is the state of a snapshot's verification. It starts as
// soon as the snapshot's integrity is first asked for and runs to the end
// in the background, so a report is "running" until every file was read.
type IntegrityReport struct {
	Status    string             `json:"status"` // running, done or failed
	Checked   uint64             `json:"checked"`
	Corrupted []IntegrityFailure `json:"corrupted"`
	Message   string             `json:"message,omitempty"`
}

type integrityRun struct {
	mu     sync.Mutex
	report IntegrityReport
}

func (run *integrityRun) snapshot() IntegrityReport {
	run.mu.Lock()
	defer run.mu.Unlock()
	report := run.report
	report.Corrupted = append([]IntegrityFailure(nil), run.report.Corrupted...)
	if report.Corrupted == nil {
		report.Corrupted = []IntegrityFailure{}
	}
	return report
}

// snapshotIntegrity reports which files of a snapshot are corrupted. The
// first call starts a verification of the whole snapshot, which reads every
// chunk; later calls return its progress, then its result.
func (ui *uiserver) snapshotIntegrity(w http.ResponseWriter, r *http.Request) error {
	snapshotID32, err := PathParamToID(r, "snapshot")
	if err != nil {
		return err
	}

	ui.integrityMu.Lock()
	run, ok := ui.integrity[snapshotID32]
	if !ok {
		run = &integrityRun{report: IntegrityReport{Status: "running"}}
		ui.integrity[snapshotID32] = run
		go ui.verifySnapshot(snapshotID32, run)
	}
	ui.integrityMu.Unlock()

	return json.NewEncoder(w).Encode(Item[IntegrityReport]{Item: run.snapshot()})
}

func (ui *uiserver) verifySnapshot(id objects.MAC, run *integrityRun) {
	err := ui.walkAndVerify(id, run)

	run.mu.Lock()
	defer run.mu.Unlock()
	if err != nil {
		run.report.Status = "failed"
		run.report.Message = err.Error()
		return
	}
	run.report.Status = "done"
}

func (ui *uiserver) walkAndVerify(id objects.MAC, run *integrityRun) error {
	snap, err := snapshot.Load(ui.repository, id)
	if err != nil {
		return fmt.Errorf("load snapshot: %w", err)
	}
	defer snap.Close()

	fs, err := snap.Filesystem()
	if err != nil {
		return fmt.Errorf("open filesystem: %w", err)
	}

	record := func(path string, failure error) {
		run.mu.Lock()
		defer run.mu.Unlock()
		run.report.Corrupted = append(run.report.Corrupted, IntegrityFailure{
			Path:  path,
			Error: failure.Error(),
		})
	}

	return fs.WalkDir("/", func(path string, entry *vfs.Entry, err error) error {
		// The server going away ends the walk; a bad entry does not.
		if err := ui.ctx.Err(); err != nil {
			return err
		}
		if err != nil {
			record(path, err)
			return nil
		}
		if !entry.Stat().Mode().IsRegular() {
			return nil
		}
		if err := ui.verifyFile(snap, entry); err != nil {
			record(path, err)
		}
		run.mu.Lock()
		run.report.Checked++
		run.mu.Unlock()
		return nil
	})
}

// verifyFile reads every chunk of a file back and recomputes its MACs, the
// way plakar check does without its cache.
func (ui *uiserver) verifyFile(snap *snapshot.Snapshot, entry *vfs.Entry) error {
	object, err := snap.LookupObject(entry.Object)
	if err != nil {
		return fmt.Errorf("object %x is missing: %w", entry.Object, err)
	}

	hasher := ui.repository.GetMACHasher()
	for _, chunk := range object.Chunks {
		data, err := ui.repository.GetBlobBytes(resources.RT_CHUNK, chunk.ContentMAC)
		if err != nil {
			return fmt.Errorf("chunk %x is unreadable: %w", chunk.ContentMAC, err)
		}
		if mac := ui.repository.ComputeMAC(data); !bytes.Equal(mac[:], chunk.ContentMAC[:]) {
			return fmt.Errorf("chunk %x is corrupted", chunk.ContentMAC)
		}
		hasher.Write(data)
	}
	if !bytes.Equal(hasher.Sum(nil), object.ContentMAC[:]) {
		return fmt.Errorf("object %x is corrupted", object.ContentMAC)
	}
	return nil
}
