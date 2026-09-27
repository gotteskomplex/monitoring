-- +goose Up
-- Time series live in TimescaleDB hypertables (ADR-0004, ADR-0010).
--
-- Security: TimescaleDB copies the hypertable's GRANTs to every chunk, but
-- NOT its row level security. A role with direct privileges on a hypertable
-- could therefore read other tenants' rows by querying a chunk directly.
-- Hypertables thus live in schema "ts", on which mon_app has no privileges.
-- mon_app accesses them only through same-named views in "public". The views
-- are owned by mon_migrator, which is subject to the (FORCEd) RLS policies,
-- and app.tenant_ids is a per-transaction setting, so the tenant filter still
-- applies to the caller. mon_system (BYPASSRLS) gets direct access to "ts".
--
-- No foreign keys from hypertables to regular tables (insert throughput);
-- the ingest path validates check ownership against the sending satellite.
-- Status values are the protobuf enum values (1=OK, 2=WARNING, 3=CRITICAL, 4=UNKNOWN).

CREATE SCHEMA ts;
GRANT USAGE ON SCHEMA ts TO mon_system;

CREATE TABLE ts.check_results (
  time         timestamptz NOT NULL,
  tenant_id    uuid NOT NULL,
  check_id     uuid NOT NULL,
  satellite_id uuid NOT NULL,
  status       smallint NOT NULL CHECK (status BETWEEN 1 AND 4),
  duration_ms  integer NOT NULL DEFAULT 0,
  attempt      smallint NOT NULL DEFAULT 1,
  message      text NOT NULL DEFAULT '',
  received_at  timestamptz NOT NULL DEFAULT now()
);
SELECT create_hypertable('ts.check_results', by_range('time', INTERVAL '1 day'));
CREATE UNIQUE INDEX check_results_uniq ON ts.check_results (tenant_id, check_id, time DESC);
SELECT enable_tenant_rls('ts.check_results');
GRANT SELECT, INSERT, UPDATE, DELETE ON ts.check_results TO mon_system;
CREATE VIEW public.check_results AS SELECT * FROM ts.check_results;

CREATE TABLE metric_series (
  id         bigint GENERATED ALWAYS AS IDENTITY,
  tenant_id  uuid NOT NULL,
  check_id   uuid NOT NULL,
  name       text NOT NULL CHECK (length(name) BETWEEN 1 AND 128),
  unit       text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (id),
  UNIQUE (tenant_id, id),
  UNIQUE (tenant_id, check_id, name),
  FOREIGN KEY (tenant_id, check_id) REFERENCES checks (tenant_id, id) ON DELETE CASCADE
);
SELECT enable_tenant_rls('metric_series');

CREATE TABLE ts.metric_points (
  time      timestamptz NOT NULL,
  tenant_id uuid NOT NULL,
  series_id bigint NOT NULL,
  value     double precision NOT NULL
);
SELECT create_hypertable('ts.metric_points', by_range('time', INTERVAL '1 day'));
CREATE UNIQUE INDEX metric_points_uniq ON ts.metric_points (tenant_id, series_id, time DESC);
SELECT enable_tenant_rls('ts.metric_points');
GRANT SELECT, INSERT, UPDATE, DELETE ON ts.metric_points TO mon_system;
CREATE VIEW public.metric_points AS SELECT * FROM ts.metric_points;

-- +goose Down
DROP VIEW public.metric_points;
DROP TABLE ts.metric_points;
DROP TABLE metric_series;
DROP VIEW public.check_results;
DROP TABLE ts.check_results;
DROP SCHEMA ts;
