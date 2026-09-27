// Package api implements the master's HTTP surface: REST API (generated from
// api/openapi.yaml), health/readiness probes, Prometheus metrics and the web UI.
package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	apigen "github.com/gotteskomplex/monitoring/internal/gen/api"
	"github.com/gotteskomplex/monitoring/internal/platform/version"
)

// Database is the subset of store.DB the HTTP layer needs for probes.
type Database interface {
	Ping(ctx context.Context) error
	SchemaVersion(ctx context.Context) (int64, error)
}

// Server wires all HTTP handlers.
type Server struct {
	db            Database
	schemaVersion int64 // latest migration embedded in the binary
	log           *slog.Logger
	registry      *prometheus.Registry
	webUI         http.Handler
}

// NewServer creates the HTTP server. webUI may be nil.
func NewServer(db Database, schemaVersion int64, log *slog.Logger, registry *prometheus.Registry, webUI http.Handler) *Server {
	return &Server{db: db, schemaVersion: schemaVersion, log: log, registry: registry, webUI: webUI}
}

// Handler returns the root handler including all middleware.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.healthz)
	mux.HandleFunc("GET /readyz", s.readyz)
	mux.Handle("GET /metrics", promhttp.HandlerFor(s.registry, promhttp.HandlerOpts{Registry: s.registry}))

	apiMux := http.NewServeMux()
	apigen.HandlerWithOptions(s, apigen.StdHTTPServerOptions{BaseURL: "/api/v1", BaseRouter: apiMux})
	mux.Handle("/api/", apiMux)

	if s.webUI != nil {
		mux.Handle("/", s.webUI)
	}
	return recoverer(s.log, securityHeaders(requestLogger(s.log, mux)))
}

var _ apigen.ServerInterface = (*Server)(nil)

// GetVersion implements GET /api/v1/version.
func (s *Server) GetVersion(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, apigen.VersionInfo{
		Version:         version.Version,
		Commit:          version.Commit,
		BuildDate:       version.Date,
		ProtocolVersion: version.ProtocolVersion,
		SchemaVersion:   s.schemaVersion,
	})
}

func (s *Server) healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// readyz reports ready only if the database is reachable and the schema is
// exactly at the version this binary expects.
func (s *Server) readyz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := s.db.Ping(ctx); err != nil {
		s.log.WarnContext(ctx, "readiness: database unreachable", slog.Any("error", err))
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unavailable", "reason": "database"})
		return
	}
	v, err := s.db.SchemaVersion(ctx)
	if err != nil || v != s.schemaVersion {
		s.log.WarnContext(ctx, "readiness: schema version mismatch",
			slog.Int64("have", v), slog.Int64("want", s.schemaVersion), slog.Any("error", err))
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unavailable", "reason": "schema"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
