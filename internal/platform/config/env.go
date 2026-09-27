// Package config reads configuration from environment variables (12-factor).
// Every variable NAME can alternatively be provided as NAME_FILE pointing to a
// file (Docker/Kubernetes secrets); the file content wins over NAME.
package config

import (
	"fmt"
	"os"
	"strings"
)

// Lookup returns the value of key, reading key_FILE if set, else def.
func Lookup(key, def string) (string, error) {
	if path := os.Getenv(key + "_FILE"); path != "" {
		b, err := os.ReadFile(path) //nolint:gosec // path is operator-provided configuration
		if err != nil {
			return "", fmt.Errorf("read %s_FILE: %w", key, err)
		}
		return strings.TrimRight(string(b), "\r\n"), nil
	}
	if v, ok := os.LookupEnv(key); ok {
		return v, nil
	}
	return def, nil
}

// Require is like Lookup but fails if the value is empty.
func Require(key string) (string, error) {
	v, err := Lookup(key, "")
	if err != nil {
		return "", err
	}
	if v == "" {
		return "", fmt.Errorf("environment variable %s (or %s_FILE) is required", key, key)
	}
	return v, nil
}
