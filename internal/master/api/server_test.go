package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"

	apigen "github.com/gotteskomplex/monitoring/internal/gen/api"
)

type fakeDB struct {
	pingErr error
	version int64
}

func (f fakeDB) Ping(context.Context) error                   { return f.pingErr }
func (f fakeDB) SchemaVersion(context.Context) (int64, error) { return f.version, nil }

func newTestServer(db Database) http.Handler {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewServer(db, 3, log, prometheus.NewRegistry(), nil).Handler()
}

func TestReadyz(t *testing.T) {
	tests := []struct {
		name string
		db   fakeDB
		want int
	}{
		{"ready", fakeDB{version: 3}, http.StatusOK},
		{"db down", fakeDB{pingErr: errors.New("down"), version: 3}, http.StatusServiceUnavailable},
		{"schema behind", fakeDB{version: 2}, http.StatusServiceUnavailable},
		{"schema ahead", fakeDB{version: 4}, http.StatusServiceUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			newTestServer(tt.db).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
			if rec.Code != tt.want {
				t.Fatalf("got %d, want %d", rec.Code, tt.want)
			}
		})
	}
}

func TestHealthzAndSecurityHeaders(t *testing.T) {
	rec := httptest.NewRecorder()
	newTestServer(fakeDB{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("healthz: got %d", rec.Code)
	}
	for _, h := range []string{"Content-Security-Policy", "Strict-Transport-Security", "X-Content-Type-Options", "X-Frame-Options", "Referrer-Policy"} {
		if rec.Header().Get(h) == "" {
			t.Errorf("missing security header %s", h)
		}
	}
}

func TestGetVersion(t *testing.T) {
	rec := httptest.NewRecorder()
	newTestServer(fakeDB{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/version", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d", rec.Code)
	}
	var v apigen.VersionInfo
	if err := json.NewDecoder(rec.Body).Decode(&v); err != nil {
		t.Fatal(err)
	}
	if v.SchemaVersion != 3 || v.ProtocolVersion != 1 {
		t.Fatalf("unexpected version info: %+v", v)
	}
}

func TestMetricsEndpoint(t *testing.T) {
	rec := httptest.NewRecorder()
	newTestServer(fakeDB{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d", rec.Code)
	}
}
