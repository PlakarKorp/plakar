package utils

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

const unableToOpenExcludesFileError = "unable to open excludes file: %w"

func LoadIgnoreFile(filename string) ([]string, error) {
	fp, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf(unableToOpenExcludesFileError, err)
	}
	defer fp.Close()

	var lines []string
	scanner := bufio.NewScanner(fp)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") {
			continue
		}
		if strings.Trim(line, " \t\r") == "" {
			continue
		}
		lines = append(lines, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}
