//go:build integration

package integration

import (
	"context"
	"database/sql"
	"testing"

	"github.com/pressly/goose/v3"

	"github.com/gotteskomplex/monitoring/internal/store"
	"github.com/gotteskomplex/monitoring/internal/testutil/pgtest"
	"github.com/gotteskomplex/monitoring/migrations"
)

// TestMigrationsRoundTrip applies all migrations, rolls them back completely
// and applies them again. Uses its own container because it drops the schema.
func TestMigrationsRoundTrip(t *testing.T) {
	ctx := context.Background()
	e, err := pgtest.Start(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = e.Terminate(ctx) }()

	sqlDB, err := sql.Open("pgx", e.MigratorDSN)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sqlDB.Close() }()
	p, err := goose.NewProvider(goose.DialectPostgres, sqlDB, migrations.FS)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.DownTo(ctx, 0); err != nil {
		t.Fatalf("down: %v", err)
	}
	if _, err := p.Up(ctx); err != nil {
		t.Fatalf("up again: %v", err)
	}
	latest, err := store.LatestMigration()
	if err != nil {
		t.Fatal(err)
	}
	got, err := p.GetDBVersion(ctx)
	if err != nil || got != latest {
		t.Fatalf("version %d (err %v), want %d", got, err, latest)
	}
	// Bootstrap must be idempotent.
	if err := store.Bootstrap(ctx, e.SuperDSN, store.RolePasswords{Migrator: "migrator-pw", App: "app-pw", System: "system-pw"}); err != nil {
		t.Fatalf("second bootstrap: %v", err)
	}
}
