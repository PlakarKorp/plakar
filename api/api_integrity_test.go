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
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSnapshotIntegrity(t *testing.T) {
	mux, _, snap, _ := server(t, "")
	defer snap.Close()
	id := snapid(snap)

	t.Run("bad id", func(t *testing.T) {
		w := get(t, mux, "/api/snapshot/integrity/zz")
		require.Equal(t, http.StatusBadRequest, w.Code, "body=%s", w.Body.String())
	})

	t.Run("intact snapshot", func(t *testing.T) {
		var report Item[IntegrityReport]
		deadline := time.Now().Add(10 * time.Second)
		for {
			w := get(t, mux, "/api/snapshot/integrity/"+id)
			require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &report))
			if report.Item.Status != "running" {
				break
			}
			require.True(t, time.Now().Before(deadline), "verification did not finish")
			time.Sleep(20 * time.Millisecond)
		}
		require.Equal(t, "done", report.Item.Status, "message=%s", report.Item.Message)
		require.Empty(t, report.Item.Corrupted)
		require.Equal(t, uint64(5), report.Item.Checked)
	})
}
