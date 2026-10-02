//go:build !windows && !darwin

package utils

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
)

var errNoGraphicalSession = errors.New("no graphical session (DISPLAY and WAYLAND_DISPLAY are not set)")

// noGraphicalSession reports whether there is no desktop to open a browser
// on. Without one, xdg-open may start a text mode browser on the terminal
// that is running plakar. WSL is exempt as it can open the Windows browser
// without a display.
func noGraphicalSession(getenv func(string) string) error {
	if getenv("DISPLAY") != "" || getenv("WAYLAND_DISPLAY") != "" || getenv("WSL_DISTRO_NAME") != "" {
		return nil
	}
	return errNoGraphicalSession
}

func openBrowser(url string) error {
	if err := noGraphicalSession(os.Getenv); err != nil {
		return err
	}

	if err := exec.Command("xdg-open", url).Start(); err != nil {
		// Try known browsers
		fallback := []string{"firefox", "chromium", "google-chrome", "chrome", "brave", "vivaldi", "opera"}
		for _, browser := range fallback {
			if err := exec.Command(browser, url).Start(); err == nil {
				return nil
			}
		}
		return fmt.Errorf("xdg-open and browser fallback failed: %w", err)
	}
	return nil
}
