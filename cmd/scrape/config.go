package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func placesAPIKeyFromConfig(path string) (string, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	section := ""
	lines := strings.Split(string(data), "\n")
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(stripConfigComment(lines[i]))
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "["), "]"))
			continue
		}
		if section != "places_api" && section != "placesAPI" {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return "", fmt.Errorf("invalid TOML line %d", i+1)
		}
		key = strings.TrimSpace(key)
		if key != "api_key" && key != "apiKey" {
			continue
		}
		apiKey, err := parseConfigString(value)
		if err != nil {
			return "", fmt.Errorf("line %d: %w", i+1, err)
		}
		return strings.TrimSpace(apiKey), nil
	}
	return "", nil
}

func stripConfigComment(line string) string {
	inString := false
	escaped := false
	for i, r := range line {
		if escaped {
			escaped = false
			continue
		}
		if r == '\\' && inString {
			escaped = true
			continue
		}
		if r == '"' {
			inString = !inString
			continue
		}
		if r == '#' && !inString {
			return line[:i]
		}
	}
	return line
}

func parseConfigString(value string) (string, error) {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, "\"") {
		return "", fmt.Errorf("expected quoted string")
	}
	return strconv.Unquote(value)
}
