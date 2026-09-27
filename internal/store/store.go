package store

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrEmptyScope is returned when WithTenantScope is called without tenants.
var ErrEmptyScope = errors.New("store: empty tenant scope")

// DB bundles the connection pools of the two runtime roles.
type DB struct {
	app    *pgxpool.Pool // mon_app: subject to RLS
	system *pgxpool.Pool // mon_system: BYPASSRLS, platform jobs only
	log    *slog.Logger
}

// Open connects both pools. systemDSN may be empty for processes that never
// run platform jobs; WithSystem then returns an error.
func Open(ctx context.Context, appDSN, systemDSN string, log *slog.Logger) (*DB, error) {
	app, err := pgxpool.New(ctx, appDSN)
	if err != nil {
		return nil, fmt.Errorf("open app pool: %w", err)
	}
	db := &DB{app: app, log: log}
	if systemDSN != "" {
		db.system, err = pgxpool.New(ctx, systemDSN)
		if err != nil {
			app.Close()
			return nil, fmt.Errorf("open system pool: %w", err)
		}
	}
	return db, nil
}

// Close releases all connections.
func (db *DB) Close() {
	db.app.Close()
	if db.system != nil {
		db.system.Close()
	}
}

// Ping checks connectivity of the application pool.
func (db *DB) Ping(ctx context.Context) error { return db.app.Ping(ctx) }

// WithTenantScope runs fn in a transaction restricted to scope by RLS.
// The transaction is committed if fn returns nil and rolled back otherwise.
func (db *DB) WithTenantScope(ctx context.Context, scope Scope, fn func(tx pgx.Tx) error) error {
	if scope.Empty() {
		return ErrEmptyScope
	}
	return pgx.BeginFunc(ctx, db.app, func(tx pgx.Tx) error {
		// set_config(..., true) == SET LOCAL: reset automatically at transaction end,
		// so pooled connections never leak a scope.
		if _, err := tx.Exec(ctx, "SELECT set_config('app.tenant_ids', $1, true)", scope.setting()); err != nil {
			return fmt.Errorf("set tenant scope: %w", err)
		}
		return fn(tx)
	})
}

// WithSystem runs fn as mon_system, bypassing RLS. Only for platform
// operations that are cross-tenant by nature (tenant lifecycle, heartbeat
// monitoring, retention). reason is logged for traceability.
//
// Note: hypertables must be accessed via schema "ts" here; the public views
// are evaluated with the view owner's RLS and would return no rows.
func (db *DB) WithSystem(ctx context.Context, reason string, fn func(tx pgx.Tx) error) error {
	if db.system == nil {
		return errors.New("store: system pool not configured")
	}
	if db.log != nil {
		db.log.DebugContext(ctx, "system db access", slog.String("reason", reason))
	}
	return pgx.BeginFunc(ctx, db.system, fn)
}
