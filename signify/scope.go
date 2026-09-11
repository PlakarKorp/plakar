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

package signify

import (
	"errors"
	"fmt"
	"net/url"
	pathpkg "path"
	"strings"
)

// LocalScope authorises filesystem installs. A distinct token rather than a
// URL, so no URL scope can grant it by accident.
const LocalScope = "local"

var ErrInvalidScope = errors.New("invalid scope")

// Scope bounds where a trusted key may have signed. A key trusted for one
// registry must not vouch for another.
type Scope struct {
	Local bool
	URL   *url.URL // the registry base, normalised; nil when Local
}

// ParseScope parses "local" or a registry base URL.
func ParseScope(s string) (*Scope, error) {
	if s == LocalScope {
		return &Scope{Local: true}, nil
	}

	u, err := url.Parse(s)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidScope, err)
	}

	switch u.Scheme {
	case "http", "https":
	default:
		return nil, fmt.Errorf("%w: scheme %q is not http or https", ErrInvalidScope, u.Scheme)
	}

	if u.Host == "" {
		return nil, fmt.Errorf("%w: no host", ErrInvalidScope)
	}

	// Credentials are not part of the identity being matched.
	if u.User != nil {
		return nil, fmt.Errorf("%w: must not contain credentials", ErrInvalidScope)
	}

	// Neither plays any part in matching.
	if u.RawQuery != "" || u.Fragment != "" {
		return nil, fmt.Errorf("%w: must not contain a query or fragment", ErrInvalidScope)
	}

	return &Scope{URL: normalizeURL(u)}, nil
}

// Resolves dot segments, which url.Parse does not: without it an origin like
// /dist/plugins/kloset/../../../etc/ would satisfy a scope of
// /dist/plugins/kloset/ while addressing /etc/. The trailing slash lets
// matching compare whole segments.
func normalizeURL(u *url.URL) *url.URL {
	n := *u
	n.Host = strings.ToLower(n.Host)

	if port := n.Port(); port != "" {
		if (n.Scheme == "https" && port == "443") || (n.Scheme == "http" && port == "80") {
			n.Host = n.Hostname()
		}
	}

	path := n.Path
	if path == "" {
		path = "/"
	}

	trailing := strings.HasSuffix(path, "/")

	path = pathpkg.Clean(path)

	if trailing || !strings.HasSuffix(path, "/") {
		if !strings.HasSuffix(path, "/") {
			path += "/"
		}
	}

	n.Path = path
	n.RawPath = ""

	return &n
}

// Matches reports whether this scope authorises origin. Exact on scheme, host
// and port, segment-wise on path: a scope for /dist/plugins/ must not match
// /dist/plugins-evil/, which a plain string prefix would allow.
func (s *Scope) Matches(origin string) bool {
	if origin == LocalScope {
		return s.Local
	}

	if s.Local || s.URL == nil {
		return false
	}

	u, err := url.Parse(origin)
	if err != nil {
		return false
	}

	if u.User != nil {
		return false
	}

	// A percent-encoded separator makes the decoded and raw paths disagree
	// about where segments begin. No legitimate registry path needs one.
	if u.RawPath != "" && u.RawPath != u.Path {
		return false
	}

	o := normalizeURL(u)

	if !strings.EqualFold(o.Scheme, s.URL.Scheme) {
		return false
	}

	if o.Host != s.URL.Host {
		return false
	}

	// Both end in "/", so this prefix test is segment-aligned.
	return strings.HasPrefix(o.Path, s.URL.Path)
}

func (s *Scope) String() string {
	if s.Local {
		return LocalScope
	}

	if s.URL == nil {
		return ""
	}

	return s.URL.String()
}
