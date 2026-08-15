package config

import (
	"bufio"
	"errors"
	"os"
	"strconv"
	"strings"
)

// LoadDotEnv loads simple KEY=VALUE entries without replacing explicit
// environment variables. Missing .env files are intentionally ignored.
func LoadDotEnv(path string) error {
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), "\"'")
		if key != "" && os.Getenv(key) == "" {
			_ = os.Setenv(key, value)
		}
	}
	return scanner.Err()
}

func StringEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func UintEnv(key string, fallback uint64) (uint64, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	return strconv.ParseUint(value, 10, 64)
}
