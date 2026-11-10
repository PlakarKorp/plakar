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

package login

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cookies"
)

func TestRevoke(t *testing.T) {
	for _, tc := range []struct {
		name    string
		status  int
		wantErr bool
	}{
		{"revoked", http.StatusNoContent, false},
		{"ok", http.StatusOK, false},
		{"unauthorized", http.StatusUnauthorized, false},
		{"server error", http.StatusInternalServerError, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var method, path, authz string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				method, path, authz = r.Method, r.URL.Path, r.Header.Get("Authorization")
				w.WriteHeader(tc.status)
			}))
			defer srv.Close()
			t.Setenv("PLAKAR_API_URL", srv.URL)

			err := Revoke(appcontext.NewAppContext(), "the-token")
			if (err != nil) != tc.wantErr {
				t.Fatalf("Revoke err = %v, want error %v", err, tc.wantErr)
			}
			if method != http.MethodPost || path != "/v1/auth/logout" || authz != "Bearer the-token" {
				t.Errorf("request = %s %s %q, want POST /v1/auth/logout with the bearer token", method, path, authz)
			}
			if tc.wantErr && !strings.Contains(err.Error(), "request failed with status") {
				t.Errorf("err = %v, want the status", err)
			}
		})
	}
}

func TestRevokeUnreachable(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	srv.Close()
	t.Setenv("PLAKAR_API_URL", srv.URL)
	if err := Revoke(appcontext.NewAppContext(), "the-token"); err == nil {
		t.Fatal("Revoke against a closed server = nil, want an error")
	}
}

func logoutContext(t *testing.T, token string) (*appcontext.AppContext, *cookies.Manager) {
	t.Helper()
	ctx := appcontext.NewAppContext()
	jar, err := cookies.NewManager(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if token != "" {
		if err := jar.PutAuthToken(token); err != nil {
			t.Fatal(err)
		}
	}
	ctx.SetCookies(jar)
	return ctx, jar
}

func TestLogout(t *testing.T) {
	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	t.Setenv("PLAKAR_TOKEN", "")
	t.Setenv("PLAKAR_API_URL", srv.URL)

	ctx, jar := logoutContext(t, "stored-token")
	revokeErr, err := Logout(ctx)
	if revokeErr != nil || err != nil {
		t.Fatalf("Logout = %v, %v", revokeErr, err)
	}
	if len(seen) != 1 || seen[0] != "Bearer stored-token" {
		t.Errorf("revoked %q, want the stored token once", seen)
	}
	if _, err := jar.GetAuthToken(); err == nil {
		t.Error("the local copy is still there")
	}

	ctx, _ = logoutContext(t, "")
	revokeErr, err = Logout(ctx)
	if revokeErr != nil || !errors.Is(err, cookies.ErrNotLoggedIn) {
		t.Errorf("Logout when logged out = %v, %v, want nil, ErrNotLoggedIn", revokeErr, err)
	}
	if len(seen) != 1 {
		t.Errorf("revoked %q, want no further call", seen)
	}

	t.Setenv("PLAKAR_TOKEN", "env-token")
	ctx, _ = logoutContext(t, "")
	revokeErr, err = Logout(ctx)
	if revokeErr != nil || !errors.Is(err, cookies.ErrDeleteEnvToken) {
		t.Errorf("Logout with an env token = %v, %v, want nil, ErrDeleteEnvToken", revokeErr, err)
	}
	if len(seen) != 1 {
		t.Errorf("revoked %q, want no further call", seen)
	}
}
