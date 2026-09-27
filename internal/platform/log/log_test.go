package log

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestRedactsSensitiveAttributes(t *testing.T) {
	tests := []struct {
		key      string
		redacted bool
	}{
		{"password", true},
		{"db_dsn", true},
		{"enrollment_token", true},
		{"snmp_community", true},
		{"Authorization", true},
		{"tenant_id", false},
		{"satellite_id", false},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			var buf bytes.Buffer
			New(&buf, "info").Info("msg", tt.key, "value-123")
			var m map[string]any
			if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
				t.Fatalf("invalid json: %v", err)
			}
			got := m[tt.key]
			if tt.redacted && got != redacted {
				t.Errorf("%s: got %v, want redacted", tt.key, got)
			}
			if !tt.redacted && got != "value-123" {
				t.Errorf("%s: got %v, want value", tt.key, got)
			}
		})
	}
}

func TestParseLevel(t *testing.T) {
	var buf bytes.Buffer
	l := New(&buf, "warn")
	l.Info("hidden")
	if buf.Len() != 0 {
		t.Fatalf("info must be suppressed at warn level")
	}
	l.Warn("shown")
	if buf.Len() == 0 {
		t.Fatalf("warn must be logged")
	}
}
