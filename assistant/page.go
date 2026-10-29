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

package assistant

import (
	"fmt"
	"strings"
)

// maxPageField bounds each string of a page context: IDs are hex, paths
// are whatever the snapshot holds, and the model gets them verbatim.
const maxPageField = 4096

// Page describes what the web UI is showing when a message is sent, so the
// model can resolve "this snapshot" or "this directory". Fields are set as
// the page has them: a snapshot page names its snapshot and directory, and
// the file being previewed; the compare page names its two snapshots.
type Page struct {
	Name     string `json:"name"`
	Snapshot string `json:"snapshot,omitempty"`
	Path     string `json:"path,omitempty"`
	File     string `json:"file,omitempty"`
	Older    string `json:"older,omitempty"`
	Newer    string `json:"newer,omitempty"`
}

// validate refuses a page the UI could not have sent.
func (p *Page) validate() error {
	for name, value := range map[string]string{
		"name": p.Name, "snapshot": p.Snapshot, "path": p.Path,
		"file": p.File, "older": p.Older, "newer": p.Newer,
	} {
		if len(value) > maxPageField {
			return fmt.Errorf("page %s exceeds %d bytes", name, maxPageField)
		}
	}
	return nil
}

// note renders the page as the sentence appended to the user's message,
// or nothing when the page has no subject worth telling the model.
func (p *Page) note() string {
	if p == nil {
		return ""
	}
	var what string
	switch p.Name {
	case "browse":
		if p.Snapshot == "" {
			return ""
		}
		what = fmt.Sprintf("snapshot %s", p.Snapshot)
		if p.File != "" {
			what += fmt.Sprintf(", previewing the file %s", p.File)
		} else if p.Path != "" {
			what += fmt.Sprintf(", browsing the directory %s", p.Path)
		}
	case "errors":
		if p.Snapshot == "" {
			return ""
		}
		what = fmt.Sprintf("the errors of snapshot %s", p.Snapshot)
		if p.Path != "" && p.Path != "/" {
			what += fmt.Sprintf(" under %s", p.Path)
		}
	case "compare":
		switch {
		case p.Older != "" && p.Newer != "":
			what = fmt.Sprintf("the comparison of snapshot %s (older) with snapshot %s (newer)", p.Older, p.Newer)
		default:
			what = "the comparison of the newest snapshot with the one before it from the same source"
		}
		if p.Path != "" && p.Path != "/" {
			what += fmt.Sprintf(", at %s", p.Path)
		}
	case "snapshots":
		what = "the list of snapshots"
	case "analytics":
		what = "the repository analytics"
	case "integrations":
		what = "the integrations catalog"
	default:
		return ""
	}
	return fmt.Sprintf("[The web UI is showing %s.]", strings.TrimSpace(what))
}
