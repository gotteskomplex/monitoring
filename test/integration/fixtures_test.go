//go:build integration

package integration

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	dbgen "github.com/gotteskomplex/monitoring/internal/gen/db"
	"github.com/gotteskomplex/monitoring/internal/store"
)

func openDB(t *testing.T) *store.DB {
	t.Helper()
	db, err := store.Open(context.Background(), env.AppDSN, env.SystemDSN, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(db.Close)
	return db
}

// tenantFixture is a tenant with one of each inventory object and results.
type tenantFixture struct {
	Tenant    dbgen.Tenant
	Site      dbgen.Site
	Satellite dbgen.Satellite
	Host      dbgen.Host
	Check     dbgen.Check
}

func newID() uuid.UUID { return uuid.Must(uuid.NewV7()) }

func createTenant(t *testing.T, db *store.DB, name string) tenantFixture {
	t.Helper()
	ctx := context.Background()
	var f tenantFixture
	id := newID()
	err := db.WithSystem(ctx, "test: create tenant", func(tx pgx.Tx) error {
		var err error
		f.Tenant, err = dbgen.New(tx).CreateTenant(ctx, dbgen.CreateTenantParams{
			ID: id, Slug: "t-" + id.String()[24:], Name: name,
		})
		return err
	})
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}

	err = db.WithTenantScope(ctx, store.TenantScope(id), func(tx pgx.Tx) error {
		q := dbgen.New(tx)
		var err error
		if f.Site, err = q.CreateSite(ctx, dbgen.CreateSiteParams{ID: newID(), TenantID: id, Name: name + " HQ", Timezone: "Europe/Berlin"}); err != nil {
			return err
		}
		if f.Satellite, err = q.CreateSatellite(ctx, dbgen.CreateSatelliteParams{ID: newID(), TenantID: id, SiteID: f.Site.ID, Name: "sat-1"}); err != nil {
			return err
		}
		if f.Host, err = q.CreateHost(ctx, dbgen.CreateHostParams{
			ID: newID(), TenantID: id, SiteID: f.Site.ID, SatelliteID: f.Satellite.ID,
			Name: "fw-" + name, Address: "10.0.0.1", Tags: []string{"firewall"},
		}); err != nil {
			return err
		}
		if f.Check, err = q.CreateCheck(ctx, dbgen.CreateCheckParams{
			ID: newID(), TenantID: id, HostID: f.Host.ID, Name: "ping", Type: "ping",
			Spec: []byte(`{"count":3}`), IntervalS: 60, TimeoutS: 5,
		}); err != nil {
			return err
		}
		now := time.Now().UTC().Truncate(time.Millisecond)
		for i := range 3 {
			if _, err = q.InsertCheckResult(ctx, dbgen.InsertCheckResultParams{
				// Spread over several days so that multiple chunks exist.
				Time: now.Add(-time.Duration(i) * 36 * time.Hour), TenantID: id, CheckID: f.Check.ID,
				SatelliteID: f.Satellite.ID, Status: 1, DurationMs: 12, Attempt: 1, Message: "OK - rtt 1ms",
			}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("create tenant objects: %v", err)
	}
	return f
}
