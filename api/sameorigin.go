package api

import (
	"mime"
	"net/http"
	"net/url"
	"strings"
)

// SameOriginMiddleware protects the mutation routes against cross-site
// requests. A browser can send a form POST to a local server from any page,
// so requests that change state must come from the server's own origin and
// must carry JSON, which a cross-site form cannot send. Requests without
// Sec-Fetch-Site and Origin come from non-browser clients and are let through.
func SameOriginMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}

		if site := r.Header.Get("Sec-Fetch-Site"); site != "" && site != "same-origin" && site != "none" {
			http.Error(w, "cross-site request refused", http.StatusForbidden)
			return
		}

		if origin := r.Header.Get("Origin"); origin != "" {
			u, err := url.Parse(origin)
			if err != nil || !strings.EqualFold(u.Host, r.Host) {
				http.Error(w, "cross-origin request refused", http.StatusForbidden)
				return
			}
		}

		if r.ContentLength != 0 {
			mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
			if err != nil || mt != "application/json" {
				http.Error(w, "content type must be application/json", http.StatusUnsupportedMediaType)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}
