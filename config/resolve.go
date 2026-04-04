package config

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

func getPassphraseFromCommand(cmd string) (string, error) {
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

func resolve(conf map[string]string) (map[string]string, error) {
	ret := make(map[string]string, len(conf))

	for k, v := range conf {
		if r, ok := strings.CutPrefix(v, "env:"); ok {
			v, ok := os.LookupEnv(r)
			if !ok {
				return nil, fmt.Errorf("env variable %q not defined", r)
			}
			ret[k] = v
		} else if r, ok := strings.CutPrefix(v, "file:"); ok {
			v, err := os.ReadFile(r)
			if err != nil {
				return nil, fmt.Errorf("failed to read %q: %w", r, err)
			}
			ret[k] = strings.TrimRight(string(v), "\r\n")
		} else if r, ok := strings.CutPrefix(v, "cmd:"); ok {
			v, err := getPassphraseFromCommand(r)
			if err != nil {
				return nil, err
			}
			ret[k] = v
		} else if k == "passphrase_cmd" {
			// XXX we used to use this field before we introduced the
			// `cmd:' prefix, need to keep the compatibility.
			v, err := getPassphraseFromCommand(v)
			if err != nil {
				return nil, err
			}
			ret["passphrase"] = v
		} else {
			// "raw:" is optional and implied
			ret[k], _ = strings.CutPrefix(v, "raw:")
		}
	}

	return ret, nil
}
