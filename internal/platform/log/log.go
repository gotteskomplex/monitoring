// Package log configures the structured JSON logger used by all binaries.
package log

import (
	"io"
	"log/slog"
	"strings"
)

// redactedKeys are attribute keys whose values are never written to logs.
var redactedKeys = []string{"password", "passwd", "secret", "token", "dsn", "authorization", "cookie", "community", "passphrase", "private_key"}

const redacted = "[REDACTED]"

// New returns a JSON logger writing to w at the given level
// ("debug", "info", "warn", "error"; default info).
func New(w io.Writer, level string) *slog.Logger {
	return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level:       parseLevel(level),
		ReplaceAttr: redact,
	}))
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func redact(_ []string, a slog.Attr) slog.Attr {
	if IsSensitiveKey(a.Key) {
		return slog.String(a.Key, redacted)
	}
	return a
}

// IsSensitiveKey reports whether a log attribute or field name likely holds a secret.
func IsSensitiveKey(key string) bool {
	k := strings.ToLower(key)
	for _, s := range redactedKeys {
		if strings.Contains(k, s) {
			return true
		}
	}
	return false
}
