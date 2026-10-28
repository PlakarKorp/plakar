package api

import (
	"net"
	"net/http"
	"strings"
)

// HostMiddleware rejects requests whose Host header is not the server's own.
// DNS rebinding makes a browser send attacker-chosen hostnames to a local
// server, so only localhost, IP literals and the host of addr are served.
// An IP literal cannot come from a rebinding page, which needs a hostname.
func HostMiddleware(addr string) func(http.Handler) http.Handler {
	configured := hostName(addr)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			host := hostName(r.Host)
			if host == "" || !(host == "localhost" || net.ParseIP(host) != nil || host == configured) {
				http.Error(w, "forbidden host", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// hostName returns the lowercased host of a host or host:port string.
func hostName(hostport string) string {
	host, _, err := net.SplitHostPort(hostport)
	if err != nil {
		host = hostport
	}
	host = strings.TrimSuffix(strings.TrimPrefix(host, "["), "]")
	return strings.ToLower(strings.TrimSuffix(host, "."))
}
