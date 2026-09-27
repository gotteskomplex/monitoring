// Package pgtest starts a TimescaleDB container for integration tests,
// bootstraps roles and applies all migrations exactly like production.
package pgtest

import (
	"context"
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/gotteskomplex/monitoring/internal/store"
)

// Image is the TimescaleDB image used in tests and docker compose.
const Image = "timescale/timescaledb:2.17.2-pg16"

var passwords = store.RolePasswords{Migrator: "migrator-pw", App: "app-pw", System: "system-pw"}

// Env is a running, migrated database.
type Env struct {
	Container   *postgres.PostgresContainer
	SuperDSN    string
	MigratorDSN string
	AppDSN      string
	SystemDSN   string
}

// Start launches the container, bootstraps and migrates. Call Terminate when done.
func Start(ctx context.Context) (*Env, error) {
	c, err := postgres.Run(ctx, Image,
		postgres.WithDatabase("monitoring"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(90*time.Second)),
	)
	if err != nil {
		return nil, fmt.Errorf("start container: %w", err)
	}
	super, err := c.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		_ = c.Terminate(ctx)
		return nil, err
	}
	e := &Env{
		Container:   c,
		SuperDSN:    super,
		MigratorDSN: withUser(super, "mon_migrator", passwords.Migrator),
		AppDSN:      withUser(super, "mon_app", passwords.App),
		SystemDSN:   withUser(super, "mon_system", passwords.System),
	}
	if err := store.Bootstrap(ctx, super, passwords); err != nil {
		_ = c.Terminate(ctx)
		return nil, err
	}
	if _, err := store.Migrate(ctx, e.MigratorDSN); err != nil {
		_ = c.Terminate(ctx)
		return nil, err
	}
	return e, nil
}

// Terminate stops the container.
func (e *Env) Terminate(ctx context.Context) error { return e.Container.Terminate(ctx) }

// Pool opens a pool for dsn that is closed at test cleanup.
func Pool(t testing.TB, dsn string) *pgxpool.Pool {
	t.Helper()
	p, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("open pool: %v", err)
	}
	t.Cleanup(p.Close)
	return p
}

func withUser(dsn, user, password string) string {
	u, err := url.Parse(dsn)
	if err != nil {
		panic(err)
	}
	u.User = url.UserPassword(user, password)
	return u.String()
}
