package utils

import (
	"fmt"
	"net/url"
)

// BrowserTrySpawn opens rawURL in the user's browser. It returns an error when
// no browser could be started, so that callers can show the URL instead.
// Only http and https URLs are opened: the URL can come from an API response
// and is handed to a launcher that would treat other schemes, or a leading
// dash, as something else.
func BrowserTrySpawn(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("refusing to open %q: not an http(s) URL", rawURL)
	}
	return openBrowser(rawURL)
}
