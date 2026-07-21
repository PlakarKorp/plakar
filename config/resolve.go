package config

import (
	"bufio"
	"fmt"
	"io"
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

func resolve(conf map[string]string, rootOverride string) (map[string]string, error) {
	ret := make(map[string]string, len(conf))

	for k, v := range conf {
		if k == "passphrase_cmd" {
			v, err := getPassphraseFromCommand(v)
			if err != nil {
				return nil, err
			}
			ret["passphrase"] = v
		} else {
			ret[k], _ = strings.CutPrefix(v, "raw:")
		}
	}

	loc, err := applyRootOverride(ret["location"], rootOverride)
	if err != nil {
		return nil, err
	}
	ret["location"] = loc

	return ret, nil
}
