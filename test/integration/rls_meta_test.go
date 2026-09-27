//go:build integration

package integration

import (
	"context"
	"slices"
	"testing"
)

// tablesWithoutTenantID lists the only tables allowed to have no tenant_id column.
var tablesWithoutTenantID = []string{
	"public.goose_db_version", // migration bookkeeping, read-only for mon_app
	"public.tenants",          // the tenant row itself; has its own RLS policies (id-based)
}

// TestEveryTenantTableHasForcedRLS is the meta-test from ARCHITECTURE.md §6.2:
// it fails as soon as a table is added without row level security.
func TestEveryTenantTableHasForcedRLS(t *testing.T) {
	ctx := context.Background()
	pool := openSuperPool(t)

	rows, err := pool.Query(ctx, `
		SELECT n.nspname || '.' || c.relname,
		       c.relrowsecurity, c.relforcerowsecurity,
		       EXISTS (SELECT 1 FROM pg_attribute a
		               WHERE a.attrelid = c.oid AND a.attname = 'tenant_id' AND NOT a.attisdropped),
		       (SELECT count(*) FROM pg_policy p WHERE p.polrelid = c.oid)
		FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname IN ('public', 'ts') AND c.relkind IN ('r', 'p')`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	seen := 0
	for rows.Next() {
		var name string
		var rls, force, hasTenant bool
		var policies int
		if err := rows.Scan(&name, &rls, &force, &hasTenant, &policies); err != nil {
			t.Fatal(err)
		}
		seen++
		if !hasTenant && !slices.Contains(tablesWithoutTenantID, name) {
			t.Errorf("%s: has no tenant_id column and is not allow-listed", name)
			continue
		}
		if name == "public.goose_db_version" {
			continue
		}
		if !rls || !force || policies == 0 {
			t.Errorf("%s: RLS enabled=%v forced=%v policies=%d; want enabled, forced, >=1 policy", name, rls, force, policies)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if seen < 10 {
		t.Fatalf("only %d tables inspected, schema not migrated?", seen)
	}
}

// TestAppRoleCannotReachHypertablesOrChunks guards against the TimescaleDB
// pitfall that chunks inherit GRANTs but not RLS (ADR-0010).
func TestAppRoleCannotReachHypertablesOrChunks(t *testing.T) {
	ctx := context.Background()
	db := openDB(t)
	createTenant(t, db, "ChunkCo") // ensures chunks exist
	pool := openSuperPool(t)

	var tsUsage bool
	if err := pool.QueryRow(ctx, `SELECT has_schema_privilege('mon_app', 'ts', 'USAGE')`).Scan(&tsUsage); err != nil {
		t.Fatal(err)
	}
	if tsUsage {
		t.Error("mon_app must not have USAGE on schema ts")
	}

	rows, err := pool.Query(ctx, `
		SELECT format('%I.%I', chunk_schema, chunk_name),
		       has_table_privilege('mon_app', format('%I.%I', chunk_schema, chunk_name)::regclass, 'SELECT,INSERT,UPDATE,DELETE')
		FROM timescaledb_information.chunks`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	chunks := 0
	for rows.Next() {
		var name string
		var priv bool
		if err := rows.Scan(&name, &priv); err != nil {
			t.Fatal(err)
		}
		chunks++
		if priv {
			t.Errorf("mon_app has privileges on chunk %s (bypasses RLS)", name)
		}
	}
	if chunks == 0 {
		t.Fatal("expected chunks to exist")
	}
}

// TestAppRoleCannotEscalate verifies mon_app cannot switch RLS off.
func TestAppRoleCannotEscalate(t *testing.T) {
	ctx := context.Background()
	app := openAppPool(t)

	var bypass, super bool
	if err := app.QueryRow(ctx, `SELECT rolbypassrls, rolsuper FROM pg_roles WHERE rolname = current_user`).Scan(&bypass, &super); err != nil {
		t.Fatal(err)
	}
	if bypass || super {
		t.Fatalf("mon_app bypassrls=%v superuser=%v", bypass, super)
	}

	for _, stmt := range []string{
		`ALTER TABLE hosts DISABLE ROW LEVEL SECURITY`,
		`ALTER TABLE hosts NO FORCE ROW LEVEL SECURITY`,
		`DROP POLICY tenant_isolation ON hosts`,
		`SET ROLE mon_system`,
		`SET ROLE mon_migrator`,
		`SELECT * FROM ts.check_results`,
		`INSERT INTO tenants (id, slug, name) VALUES (gen_random_uuid(), 'evil', 'Evil')`,
		`INSERT INTO goose_db_version (version_id, is_applied) VALUES (999, true)`,
	} {
		if _, err := app.Exec(ctx, stmt); err == nil {
			t.Errorf("mon_app could execute %q", stmt)
		}
	}

	// With row_security=off PostgreSQL must refuse instead of silently filtering.
	tx, err := app.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SET LOCAL row_security = off`); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `SELECT count(*) FROM hosts`); err == nil {
		t.Error("query with row_security=off must fail for mon_app")
	}
}
