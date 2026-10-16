package web

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestCorsMiddleware_SetsHeadersOnGET(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		_, _ = io.WriteString(w, "ok")
	})

	handler := corsMiddleware(next)

	req := httptest.NewRequest(http.MethodGet, "/anything", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Fatal("next handler should have been called for GET")
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("Allow-Origin = %q, want *", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Fatal("Allow-Methods should be set")
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); got == "" {
		t.Fatal("Allow-Headers should be set")
	}
	if rec.Body.String() != "ok" {
		t.Fatalf("body = %q, want ok", rec.Body.String())
	}
}

func TestCorsMiddleware_OptionsShortCircuits(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	handler := corsMiddleware(next)

	req := httptest.NewRequest(http.MethodOptions, "/anything", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if called {
		t.Fatal("next handler must NOT be called for preflight OPTIONS")
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("Allow-Origin = %q, want *", got)
	}
}

func TestStaticHandler_ServesPlaceholderWithoutBuild(t *testing.T) {
	handler, err := staticHandler(fstest.MapFS{
		"build/.gitkeep": &fstest.MapFile{},
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, target := range []string{"/", "/snapshots"} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))

		if rec.Code != http.StatusNotFound {
			t.Fatalf("%s: status = %d, want %d", target, rec.Code, http.StatusNotFound)
		}
		if !strings.Contains(rec.Body.String(), "built without its web UI") {
			t.Fatalf("%s: body does not explain the UI is missing: %q", target, rec.Body.String())
		}
	}
}

func TestStaticHandler_ServesBuild(t *testing.T) {
	handler, err := staticHandler(fstest.MapFS{
		"build/index.html":    &fstest.MapFile{Data: []byte("index")},
		"build/assets/app.js": &fstest.MapFile{Data: []byte("app")},
	})
	if err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.Handle("/{path...}", handler)

	for target, want := range map[string]string{
		"/assets/app.js": "app",
		"/snapshots":     "index",
	} {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))

		if rec.Code != http.StatusOK {
			t.Fatalf("%s: status = %d, want %d", target, rec.Code, http.StatusOK)
		}
		if rec.Body.String() != want {
			t.Fatalf("%s: body = %q, want %q", target, rec.Body.String(), want)
		}
	}
}
