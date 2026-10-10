package utils

import (
	"fmt"

	"golang.org/x/sys/windows"
)

// openBrowser asks the shell to open the URL and reports when it cannot, for
// instance when no application handles the URL, so that the caller can print
// it instead.
func openBrowser(url string) error {
	verb, err := windows.UTF16PtrFromString("open")
	if err != nil {
		return err
	}
	file, err := windows.UTF16PtrFromString(url)
	if err != nil {
		return err
	}
	if err := windows.ShellExecute(0, verb, file, nil, nil, windows.SW_SHOWNORMAL); err != nil {
		return fmt.Errorf("open %s: %w", url, err)
	}
	return nil
}
