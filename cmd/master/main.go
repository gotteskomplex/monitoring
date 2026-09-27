// Command master runs the central monitoring server.
//
// Usage:
//
//	master serve       run HTTP API / web UI (gRPC gateway follows in phase 2)
//	master migrate     apply database migrations (as mon_migrator)
//	master bootstrap   create DB roles and extensions (as superuser, idempotent)
//	master healthcheck probe the local /healthz endpoint (container healthcheck)
//	master version     print build information
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"

	"github.com/gotteskomplex/monitoring/internal/master/api"
	"github.com/gotteskomplex/monitoring/internal/master/webui"
	"github.com/gotteskomplex/monitoring/internal/platform/config"
	mlog "github.com/gotteskomplex/monitoring/internal/platform/log"
	"github.com/gotteskomplex/monitoring/internal/platform/version"
	"github.com/gotteskomplex/monitoring/internal/store"
)

func main() {
	level, _ := config.Lookup("MON_LOG_LEVEL", "info")
	log := mlog.New(os.Stdout, level).With(slog.String("component", "master"))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cmd := "serve"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	var err error
	switch cmd {
	case "serve":
		err = serve(ctx, log)
	case "migrate":
		err = migrate(ctx, log)
	case "bootstrap":
		err = bootstrap(ctx, log)
	case "healthcheck":
		err = healthcheck(ctx)
	case "version":
		fmt.Printf("master %s (commit %s, built %s, protocol %d)\n",
			version.Version, version.Commit, version.Date, version.ProtocolVersion)
	default:
		err = fmt.Errorf("unknown command %q (serve, migrate, bootstrap, healthcheck, version)", cmd)
	}
	if err != nil {
		log.Error("command failed", slog.String("command", cmd), slog.Any("error", err))
		os.Exit(1)
	}
}

func serve(ctx context.Context, log *slog.Logger) error {
	addr, err := config.Lookup("MON_HTTP_ADDR", ":8080")
	if err != nil {
		return err
	}
	appDSN, err := config.Require("MON_DB_APP_DSN")
	if err != nil {
		return err
	}
	systemDSN, err := config.Lookup("MON_DB_SYSTEM_DSN", "")
	if err != nil {
		return err
	}
	latest, err := store.LatestMigration()
	if err != nil {
		return err
	}

	db, err := store.Open(ctx, appDSN, systemDSN, log)
	if err != nil {
		return err
	}
	defer db.Close()

	reg := prometheus.NewRegistry()
	reg.MustRegister(collectors.NewGoCollector(), collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	srv := &http.Server{
		Addr:              addr,
		Handler:           api.NewServer(db, latest, log, reg, webui.Handler()).Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("http server listening", slog.String("addr", addr), slog.String("version", version.Version))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}
	log.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

func migrate(ctx context.Context, log *slog.Logger) error {
	dsn, err := config.Require("MON_DB_MIGRATOR_DSN")
	if err != nil {
		return err
	}
	v, err := store.Migrate(ctx, dsn)
	if err != nil {
		return err
	}
	log.Info("database migrated", slog.Int64("schema_version", v))
	return nil
}

func bootstrap(ctx context.Context, log *slog.Logger) error {
	dsn, err := config.Require("MON_DB_SUPERUSER_DSN")
	if err != nil {
		return err
	}
	var pw store.RolePasswords
	for key, dst := range map[string]*string{
		"MON_DB_MIGRATOR_PASSWORD": &pw.Migrator,
		"MON_DB_APP_PASSWORD":      &pw.App,
		"MON_DB_SYSTEM_PASSWORD":   &pw.System,
	} {
		if *dst, err = config.Require(key); err != nil {
			return err
		}
	}
	if err := store.Bootstrap(ctx, dsn, pw); err != nil {
		return err
	}
	log.Info("database bootstrapped")
	return nil
}

// healthcheck is used as container healthcheck because the runtime image has no shell or curl.
func healthcheck(ctx context.Context) error {
	addr, err := config.Lookup("MON_HTTP_ADDR", ":8080")
	if err != nil {
		return err
	}
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("parse MON_HTTP_ADDR: %w", err)
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+net.JoinHostPort(host, port)+"/healthz", nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("healthz returned %d", resp.StatusCode)
	}
	return nil
}
