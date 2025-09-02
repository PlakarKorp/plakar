package utils

import (
	"fmt"
	"os/exec"
	"runtime"
)

func BrowserTrySpawn(url string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd := exec.Command("xdg-open", url)
		if err := cmd.Start(); err != nil {
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
}
