package utils

// BrowserTrySpawn opens url in the user's browser. It returns an error when
// no browser could be started, so that callers can show the URL instead.
func BrowserTrySpawn(url string) error {
	return openBrowser(url)
}
