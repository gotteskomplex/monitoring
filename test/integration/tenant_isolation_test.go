//go:build integration

package integration

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	dbgen "github.com/gotteskomplex/monitoring/internal/gen/db"
	"github.com/gotteskomplex/monitoring/internal/store"
	"github.com/gotteskomplex/monitoring/internal/testutil/pgtest"
)

// TestTenantIsolation proves on database level that tenant A can neither read
// nor write data of tenant B. API-level isolation tests follow in phase 4,
// when the first tenant-scoped endpoints exist.
func TestTenantIsolation(t *testing.T) {
	ctx := context.Background()
	db := openDB(t)
	a := createTenant(t, db, "Alpha")
	b := createTenant(t, db, "Beta")
	scopeA := store.TenantScope(a.Tenant.ID)
	scopeB := store.TenantScope(b.Tenant.ID)

	inScope := func(t *testing.T, s store.Scope, fn func(q *dbgen.Queries) error) error {
		t.Helper()
		return db.WithTenantScope(ctx, s, func(tx pgx.Tx) error { return fn(dbgen.New(tx)) })
	}

	t.Run("A reads only its own rows", func(t *testing.T) {
		err := inScope(t, scopeA, func(q *dbgen.Queries) error {
			tenants, err := q.ListTenants(ctx)
			if err != nil {
				return err
			}
			if len(tenants) != 1 || tenants[0].ID != a.Tenant.ID {
				return fmt.Errorf("tenants visible: %+v", tenants)
			}
			if _, err := q.GetTenant(ctx, b.Tenant.ID); !errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("GetTenant(B) = %w, want ErrNoRows", err)
			}
			sites, _ := q.ListSites(ctx)
			sats, _ := q.ListSatellites(ctx)
			hosts, _ := q.ListHosts(ctx)
			checks, _ := q.ListChecks(ctx)
			for _, s := range sites {
				assertTenant(t, "site", s.TenantID, a)
			}
			for _, s := range sats {
				assertTenant(t, "satellite", s.TenantID, a)
			}
			for _, h := range hosts {
				assertTenant(t, "host", h.TenantID, a)
			}
			for _, c := range checks {
				assertTenant(t, "check", c.TenantID, a)
			}
			if len(sites) != 1 || len(sats) != 1 || len(hosts) != 1 || len(checks) != 1 {
				return fmt.Errorf("unexpected counts sites=%d sats=%d hosts=%d checks=%d", len(sites), len(sats), len(hosts), len(checks))
			}
			res, err := q.ListRecentResults(ctx, dbgen.ListRecentResultsParams{CheckID: b.Check.ID, MaxRows: 100})
			if err != nil {
				return err
			}
			if len(res) != 0 {
				return fmt.Errorf("A sees %d results of B's check", len(res))
			}
			n, err := q.CountCheckResults(ctx)
			if err != nil {
				return err
			}
			if n != 3 {
				return fmt.Errorf("A sees %d results, want 3", n)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("A cannot insert rows for B", func(t *testing.T) {
		attempts := map[string]func(q *dbgen.Queries) error{
			"site": func(q *dbgen.Queries) error {
				_, err := q.CreateSite(ctx, dbgen.CreateSiteParams{ID: newID(), TenantID: b.Tenant.ID, Name: "evil", Timezone: "UTC"})
				return err
			},
			"host": func(q *dbgen.Queries) error {
				_, err := q.CreateHost(ctx, dbgen.CreateHostParams{
					ID: newID(), TenantID: b.Tenant.ID, SiteID: b.Site.ID, SatelliteID: b.Satellite.ID,
					Name: "evil", Address: "10.6.6.6", Tags: []string{},
				})
				return err
			},
			"check result": func(q *dbgen.Queries) error {
				_, err := q.InsertCheckResult(ctx, dbgen.InsertCheckResultParams{
					Time: time.Now(), TenantID: b.Tenant.ID, CheckID: b.Check.ID, SatelliteID: a.Satellite.ID,
					Status: 3, Attempt: 1, Message: "forged",
				})
				return err
			},
		}
		for name, fn := range attempts {
			t.Run(name, func(t *testing.T) {
				if err := inScope(t, scopeA, fn); err == nil {
					t.Fatalf("insert of %s for tenant B succeeded in scope A", name)
				}
			})
		}
	})

	t.Run("A cannot update or delete B's rows", func(t *testing.T) {
		err := inScope(t, scopeA, func(q *dbgen.Queries) error {
			n, err := q.UpdateHostAddress(ctx, dbgen.UpdateHostAddressParams{ID: b.Host.ID, Address: "6.6.6.6"})
			if err != nil || n != 0 {
				return fmt.Errorf("update B host: rows=%d err=%w", n, err)
			}
			if n, err = q.DeleteHost(ctx, b.Host.ID); err != nil || n != 0 {
				return fmt.Errorf("delete B host: rows=%d err=%w", n, err)
			}
			if n, err = q.RenameTenant(ctx, dbgen.RenameTenantParams{ID: b.Tenant.ID, Name: "pwned"}); err != nil || n != 0 {
				return fmt.Errorf("rename B tenant: rows=%d err=%w", n, err)
			}
			if n, err = q.DeleteTenant(ctx, a.Tenant.ID); err != nil || n != 0 {
				return fmt.Errorf("app role deleted a tenant: rows=%d err=%w", n, err)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		// B's data is unchanged.
		err = inScope(t, scopeB, func(q *dbgen.Queries) error {
			hosts, err := q.ListHosts(ctx)
			if err != nil {
				return err
			}
			if len(hosts) != 1 || hosts[0].Address != b.Host.Address {
				return fmt.Errorf("B's host was modified: %+v", hosts)
			}
			tn, err := q.GetTenant(ctx, b.Tenant.ID)
			if err != nil || tn.Name != "Beta" {
				return fmt.Errorf("B tenant modified: %+v %w", tn, err)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("A cannot move its rows into tenant B", func(t *testing.T) {
		err := db.WithTenantScope(ctx, scopeA, func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, `UPDATE hosts SET tenant_id = $1 WHERE id = $2`, b.Tenant.ID, a.Host.ID)
			return err
		})
		if err == nil {
			t.Fatal("moving a row to another tenant must fail (RLS WITH CHECK)")
		}
	})

	t.Run("cross-tenant references are impossible even with multi-tenant scope", func(t *testing.T) {
		both := store.TenantScope(a.Tenant.ID, b.Tenant.ID)
		err := inScope(t, both, func(q *dbgen.Queries) error {
			_, err := q.CreateHost(ctx, dbgen.CreateHostParams{
				ID: newID(), TenantID: a.Tenant.ID, SiteID: b.Site.ID, SatelliteID: a.Satellite.ID,
				Name: "mixed", Address: "10.1.1.1", Tags: []string{},
			})
			return err
		})
		if err == nil {
			t.Fatal("host of tenant A referencing site of tenant B must violate the composite FK")
		}
		err = db.WithSystem(ctx, "test: cross tenant fk", func(tx pgx.Tx) error {
			_, err := dbgen.New(tx).CreateCheck(ctx, dbgen.CreateCheckParams{
				ID: newID(), TenantID: a.Tenant.ID, HostID: b.Host.ID, Name: "x", Type: "ping",
				Spec: []byte(`{}`), IntervalS: 60, TimeoutS: 5,
			})
			return err
		})
		if err == nil {
			t.Fatal("composite FK must hold even for the BYPASSRLS system role")
		}
	})

	t.Run("multi-tenant scope sees exactly its tenants", func(t *testing.T) {
		err := inScope(t, store.TenantScope(a.Tenant.ID, b.Tenant.ID), func(q *dbgen.Queries) error {
			hosts, err := q.ListHosts(ctx)
			if err != nil {
				return err
			}
			for _, h := range hosts {
				if h.TenantID != a.Tenant.ID && h.TenantID != b.Tenant.ID {
					return fmt.Errorf("foreign host visible: %v", h.TenantID)
				}
			}
			if len(hosts) != 2 {
				return fmt.Errorf("want 2 hosts, got %d", len(hosts))
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("no scope means no data (fail closed)", func(t *testing.T) {
		if err := db.WithTenantScope(ctx, store.Scope{}, func(pgx.Tx) error { return nil }); !errors.Is(err, store.ErrEmptyScope) {
			t.Fatalf("empty scope: got %v, want ErrEmptyScope", err)
		}
		app := openAppPool(t)
		for _, tbl := range []string{"tenants", "sites", "satellites", "hosts", "checks", "check_results", "check_state", "metric_series", "metric_points"} {
			var n int
			if err := app.QueryRow(ctx, "SELECT count(*) FROM "+tbl).Scan(&n); err != nil {
				t.Fatalf("%s: %v", tbl, err)
			}
			if n != 0 {
				t.Errorf("%s: %d rows visible without scope", tbl, n)
			}
		}
	})

	t.Run("scope does not leak to the next use of a pooled connection", func(t *testing.T) {
		single, err := store.Open(ctx, env.AppDSN+"&pool_max_conns=1", "", nil)
		if err != nil {
			t.Fatal(err)
		}
		defer single.Close()
		if err := single.WithTenantScope(ctx, scopeA, func(pgx.Tx) error { return nil }); err != nil {
			t.Fatal(err)
		}
		p := pgtest.Pool(t, env.AppDSN+"&pool_max_conns=1")
		var setting string
		if err := p.QueryRow(ctx, "SELECT COALESCE(current_setting('app.tenant_ids', true), '')").Scan(&setting); err != nil {
			t.Fatal(err)
		}
		if setting != "" {
			t.Fatalf("app.tenant_ids leaked: %q", setting)
		}
		var n int
		if err := single.WithTenantScope(ctx, scopeB, func(tx pgx.Tx) error {
			return tx.QueryRow(ctx, "SELECT count(*) FROM hosts WHERE tenant_id = $1", a.Tenant.ID).Scan(&n)
		}); err != nil {
			t.Fatal(err)
		}
		if n != 0 {
			t.Fatalf("scope B on reused connection sees %d hosts of A", n)
		}
	})
}

func assertTenant(t *testing.T, kind string, got interface{ String() string }, want tenantFixture) {
	t.Helper()
	if got.String() != want.Tenant.ID.String() {
		t.Errorf("%s of tenant %s visible in scope of %s", kind, got, want.Tenant.ID)
	}
}
