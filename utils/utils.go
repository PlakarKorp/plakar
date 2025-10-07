/*
 * Copyright (c) 2021 Gilles Chehade <gilles@poolp.org>
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

package utils

import (
	"bufio"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"time"
	"unicode"

	passwordvalidator "github.com/wagslane/go-password-validator"
	"golang.org/x/mod/semver"
	"golang.org/x/term"
	"golang.org/x/tools/blog/atom"
)

// ParseSnapshotID parses a string in the form snapshotID[:pattern] and
// returns the two strings.
func ParseSnapshotID(id string) (string, string) {
	tmp := strings.SplitN(id, ":", 2)
	prefix := id
	pattern := ""
	if len(tmp) == 2 {
		prefix, pattern = tmp[0], tmp[1]
		if runtime.GOOS != "windows" {
			if !strings.HasPrefix(pattern, "/") {
				pattern = "/" + pattern
			}
		}
	}
	return prefix, pattern
}

type ReleaseUpdateSummary struct {
	FoundCount int
	Latest     string

	SecurityFix    bool
	ReliabilityFix bool
}

func shouldCheckUpdate(cachedir string) bool {
	if strings.Contains(VERSION, "devel") {
		return false
	}

	cookie := path.Join(cachedir, "last-update-check")
	cutoff := time.Now().Add(-24 * time.Hour)

	sb, err := os.Stat(cookie)
	if err == nil && sb.ModTime().After(cutoff) {
		return false
	}

	file, err := os.Create(cookie)
	if err != nil {
		file.Close()
	}

	return true
}

func CheckUpdate(cachedir string) (update ReleaseUpdateSummary, err error) {
	if !shouldCheckUpdate(cachedir) {
		return
	}

	req, err := http.NewRequest("GET", "https://plakar.io/api/releases.atom", nil)
	if err != nil {
		return
	}

	req.Header.Set("User-Agent", fmt.Sprintf("plakar/%s (%s/%s)", VERSION, runtime.GOOS, runtime.GOARCH))

	client := http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return
	}
	defer res.Body.Close()

	var feed []atom.Feed
	err = xml.NewDecoder(res.Body).Decode(&feed)
	if err != nil {
		return
	}

	var latestEntry *atom.Entry
	for _, entry := range feed[0].Entry {
		if !semver.IsValid(entry.Title) {
			continue
		}
		if semver.Compare(VERSION, entry.Title) >= 0 {
			continue
		}

		update.FoundCount++

		if latestEntry == nil || semver.Compare(latestEntry.Title, entry.Title) < 0 {
			latestEntry = entry
		}

		if latestEntry.Content == nil {
			continue
		}

		body := latestEntry.Content.Body
		if strings.HasPrefix(body, "SECURITY") {
			update.SecurityFix = true
		}
		if strings.HasPrefix(body, "RELIABILITY") {
			update.ReliabilityFix = true
		}
	}

	return
}

func readpassphrase(in, out *os.File, prefix string) ([]byte, error) {
	fmt.Fprint(out, prefix)
	passphrase, err := term.ReadPassword(int(in.Fd()))
	fmt.Fprintf(os.Stderr, "\n")
	if err != nil {
		return nil, err
	}
	return passphrase, nil
}

func GetPassphrase(prefix string) ([]byte, error) {
	var in, out = os.Stdin, os.Stderr

	// use the tty for I/O if possible
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err == nil {
		in, out = tty, tty
		defer tty.Close()
	}

	return readpassphrase(in, out, prefix+" passphrase: ")
}

func GetPassphraseFromCommand(cmd string) (string, error) {
	var c *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		c = exec.Command("cmd", "/C", cmd)
	default: // assume unix-esque
		c = exec.Command("/bin/sh", "-c", cmd)
	}

	stdout, err := c.StdoutPipe()
	if err != nil {
		return "", err
	}

	if err := c.Start(); err != nil {
		return "", err
	}

	var pass string
	var lines int
	scan := bufio.NewScanner(stdout)
	for scan.Scan() {
		pass = scan.Text()
		lines++
	}

	// don't deadlock in case the scanner fails
	io.Copy(io.Discard, stdout)

	if err := c.Wait(); err != nil {
		return "", err
	}

	if err := scan.Err(); err != nil {
		return "", err
	}

	if lines != 1 {
		return "", fmt.Errorf("passphrase_cmd returned too many lines")
	}

	return pass, nil
}

func GetPassphraseConfirm(prefix string, minEntropyBits float64, retry int) ([]byte, error) {
	var in, out = os.Stdin, os.Stderr

	// use the tty for I/O if possible
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err == nil {
		in, out = tty, tty
		defer tty.Close()
	}

	var previous []byte
	first := true // avoid matching emtpty lines

	for range retry {

		passphrase1, err := readpassphrase(in, out, prefix+" passphrase: ")
		if err != nil {
			return nil, err
		}

		if first || string(passphrase1) != string(previous) {
			previous = passphrase1
			first = false
			// keepass considers < 80 bits as weak
			err = passwordvalidator.Validate(string(passphrase1), minEntropyBits)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", err)
				continue
			}
		}
		// reset our force-weak state past this point.
		previous = nil
		first = true

		passphrase2, err := readpassphrase(in, out, prefix+" passphrase (confirm): ")
		if err != nil {
			return nil, err
		}

		if string(passphrase1) != string(passphrase2) {
			fmt.Fprintf(os.Stderr, "passphrases mismatch\n")
			continue
		}

		return passphrase1, nil
	}

	return nil, errors.New("too many failed attempts")
}

func GetCacheDir(appName string) (string, error) {
	var cacheDir string

	switch runtime.GOOS {
	case "windows":
		// Use %LocalAppData%
		cacheDir = os.Getenv("LocalAppData")
		if cacheDir == "" {
			return "", fmt.Errorf("LocalAppData environment variable not set")
		}
		cacheDir = filepath.Join(cacheDir, appName)
	default:
		// Use XDG_CACHE_HOME or default to ~/.cache
		cacheDir = os.Getenv("XDG_CACHE_HOME")
		if cacheDir == "" {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			cacheDir = filepath.Join(homeDir, ".cache", appName)
		} else {
			cacheDir = filepath.Join(cacheDir, appName)
		}
	}

	// Create the cache directory if it doesn't exist
	err := os.MkdirAll(cacheDir, 0700)
	if err != nil {
		return "", err
	}

	return cacheDir, nil
}

func GetConfigDir(appName string) (string, error) {
	var configDir string

	switch runtime.GOOS {
	case "windows":
		// Use %LocalAppData%
		configDir = os.Getenv("LocalAppData")
		if configDir == "" {
			return "", fmt.Errorf("LocalAppData environment variable not set")
		}
		configDir = filepath.Join(configDir, appName)
	default:
		// Use XDG_CONFIG_HOME or default to ~/.config
		configDir = os.Getenv("XDG_CONFIG_HOME")
		if configDir == "" {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			configDir = filepath.Join(homeDir, ".config", appName)
		} else {
			configDir = filepath.Join(configDir, appName)
		}
	}

	// Create the cache directory if it doesn't exist
	err := os.MkdirAll(configDir, 0700)
	if err != nil {
		return "", err
	}

	return configDir, nil
}

func GetDataDir(appName string) (string, error) {
	var dataDir string

	switch runtime.GOOS {
	case "windows":
		// Use %LocalAppData%
		dataDir = os.Getenv("LocalAppData")
		if dataDir == "" {
			return "", fmt.Errorf("LocalAppData environment variable not set")
		}
		dataDir = filepath.Join(dataDir, appName)
	default:
		// Use XDG_DATA_HOME or default to ~/.local/share
		dataDir = os.Getenv("XDG_DATA_HOME")
		if dataDir == "" {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			dataDir = filepath.Join(homeDir, ".local", "share", appName)
		} else {
			dataDir = filepath.Join(dataDir, appName)
		}
	}

	// Create the cache directory if it doesn't exist
	err := os.MkdirAll(dataDir, 0700)
	if err != nil {
		return "", err
	}

	return dataDir, nil
}

var VERSION = func() string {
	version := "v1.0.5-devel"
	if strings.HasSuffix(version, "-devel") {
		if info, ok := debug.ReadBuildInfo(); ok {
			for _, setting := range info.Settings {
				if setting.Key == "vcs.revision" {
					version += "." + setting.Value
				}
			}
		}
	}
	return version
}()

func init() {
	if !semver.IsValid(VERSION) {
		panic(fmt.Sprintf("invalid version string: %s", VERSION))
	}
}

func GetVersion() string {
	return VERSION
}

var sbuilderPool = sync.Pool{
	New: func() any {
		return new(strings.Builder)
	},
}

func issafe(str string) bool {
	for _, r := range str {
		if !unicode.IsPrint(r) {
			return false
		}
	}
	return true
}

func SanitizeText(input string) string {
	if issafe(input) {
		return input
	}

	sb := sbuilderPool.Get().(*strings.Builder)
	defer sbuilderPool.Put(sb)
	sb.Reset()

	for _, r := range input {
		if unicode.IsPrint(r) {
			sb.WriteRune(r)
		} else {
			sb.WriteRune('?')
		}
	}

	return sb.String()
}

func ValidateEmail(email string) (string, error) {
	if email == "" {
		return "", errors.New("email cannot be empty")
	}
	mail, err := mail.ParseAddress(email)
	if err != nil {
		return "", fmt.Errorf("invalid email address: %w", err)
	}
	if mail.Address != email {
		return "", fmt.Errorf("invalid email address: %w", err)
	}
	return mail.Address, nil
}
