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

// SourceIgnoreRules returns the exclude rules a configured source carries in
// its "ignore" and "ignore-file" keys, each a comma-separated list.
func SourceIgnoreRules(source map[string]string) ([]string, error) {
	var rules []string
	for _, filename := range splitCommaList(source["ignore-file"]) {
		lines, err := LoadIgnoreFile(filename)
		if err != nil {
			return nil, err
		}
		rules = append(rules, lines...)
	}
	return append(rules, splitCommaList(source["ignore"])...), nil
}

func splitCommaList(value string) []string {
	var items []string
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			items = append(items, item)
		}
	}
	return items
}
