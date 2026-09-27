package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"

	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib" // database/sql driver for goose
	"github.com/pressly/goose/v3"

	"github.com/gotteskomplex/monitoring/migrations"
)

// RolePasswords are the passwords set for the database roles by Bootstrap.
type RolePasswords struct {
	Migrator string
	App      string
	System   string
}

// Bootstrap creates roles and extensions. It must run as a superuser against
// the target database and is idempotent.
func Bootstrap(ctx context.Context, superuserDSN string, pw RolePasswords) error {
	if pw.Migrator == "" || pw.App == "" || pw.System == "" {
		return fmt.Errorf("bootstrap: all role passwords are required")
	}
	conn, err := pgx.Connect(ctx, superuserDSN)
	if err != nil {
		return fmt.Errorf("bootstrap: connect: %w", err)
	}
	defer func() { _ = conn.Close(ctx) }()

	return pgx.BeginFunc(ctx, conn, func(tx pgx.Tx) error {
		for k, v := range map[string]string{
			"mon.migrator_password": pw.Migrator,
			"mon.app_password":      pw.App,
			"mon.system_password":   pw.System,
		} {
			if _, err := tx.Exec(ctx, "SELECT set_config($1, $2, true)", k, v); err != nil {
				return fmt.Errorf("bootstrap: set %s: %w", k, err)
			}
		}
		if _, err := tx.Exec(ctx, migrations.BootstrapSQL); err != nil {
			return fmt.Errorf("bootstrap: %w", err)
		}
		return nil
	})
}

func newProvider(db *sql.DB) (*goose.Provider, error) {
	p, err := goose.NewProvider(goose.DialectPostgres, db, migrations.FS)
	if err != nil {
		return nil, fmt.Errorf("goose provider: %w", err)
	}
	return p, nil
}

// Migrate applies all pending migrations. migratorDSN must connect as mon_migrator.
func Migrate(ctx context.Context, migratorDSN string) (int64, error) {
	db, err := sql.Open("pgx", migratorDSN)
	if err != nil {
		return 0, fmt.Errorf("migrate: open: %w", err)
	}
	defer func() { _ = db.Close() }()
	p, err := newProvider(db)
	if err != nil {
		return 0, err
	}
	if _, err := p.Up(ctx); err != nil {
		return 0, fmt.Errorf("migrate: up: %w", err)
	}
	return p.GetDBVersion(ctx)
}

// LatestMigration returns the highest migration version embedded in the binary.
func LatestMigration() (int64, error) {
	names, err := fs.Glob(migrations.FS, "*.sql")
	if err != nil {
		return 0, fmt.Errorf("list migrations: %w", err)
	}
	var latest int64
	for _, n := range names {
		v, err := goose.NumericComponent(n)
		if err != nil {
			return 0, fmt.Errorf("migration %s: %w", n, err)
		}
		latest = max(latest, v)
	}
	if latest == 0 {
		return 0, errors.New("no migrations embedded")
	}
	return latest, nil
}

// SchemaVersion returns the applied schema version as seen by the app role.
func (db *DB) SchemaVersion(ctx context.Context) (int64, error) {
	var v int64
	err := db.app.QueryRow(ctx,
		"SELECT COALESCE(max(version_id), 0) FROM goose_db_version WHERE is_applied").Scan(&v)
	if err != nil {
		return 0, fmt.Errorf("schema version: %w", err)
	}
	return v, nil
}
