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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/PlakarKorp/plakar/cookies"
	"github.com/stretchr/testify/require"
)

func TestLogoutRevokesServerSide(t *testing.T) {
	for name, status := range map[string]int{"revoked": http.StatusNoContent, "refused": http.StatusInternalServerError} {
		t.Run(name, func(t *testing.T) {
			var seen []string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				seen = append(seen, r.Method+" "+r.URL.Path+" "+r.Header.Get("Authorization"))
				w.WriteHeader(status)
			}))
			defer srv.Close()
			t.Setenv("PLAKAR_TOKEN", "")
			t.Setenv("PLAKAR_API_URL", srv.URL)

			mux, _, snap, ctx := server(t, "")
			defer snap.Close()
			jar, err := cookies.NewManager(t.TempDir())
			require.NoError(t, err)
			ctx.SetCookies(jar)
			require.NoError(t, jar.PutAuthToken("ui-token"))

			req, _ := http.NewRequest("POST", "/api/authentication/logout", nil)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)
			require.Equal(t, http.StatusOK, w.Code, w.Body.String())
			require.Equal(t, []string{"POST /v1/auth/logout Bearer ui-token"}, seen)
			_, err = jar.GetAuthToken()
			require.Error(t, err, "the local copy is gone")
		})
	}
}
