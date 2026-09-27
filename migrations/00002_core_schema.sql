-- +goose Up

-- Tenants ---------------------------------------------------------------
-- Tenants have no tenant_id column; the row itself is the tenant. The app
-- role may only read/update tenants in its scope. Creating and deleting
-- tenants is a platform operation performed with the mon_system role.
CREATE TABLE tenants (
  id                      uuid PRIMARY KEY,
  slug                    text NOT NULL UNIQUE CHECK (slug ~ '^[a-z0-9][a-z0-9-]{1,62}$'),
  name                    text NOT NULL CHECK (length(name) BETWEEN 1 AND 200),
  status                  text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'suspended', 'deleting')),
  retention_tier          text NOT NULL DEFAULT 'standard',
  customer_portal_enabled boolean NOT NULL DEFAULT false,
  settings                jsonb NOT NULL DEFAULT '{}',
  created_at              timestamptz NOT NULL DEFAULT now(),
  updated_at              timestamptz NOT NULL DEFAULT now()
);
ALTER TABLE tenants ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenants FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_read ON tenants FOR SELECT USING (id = ANY (app_tenant_ids()));
CREATE POLICY tenant_update ON tenants FOR UPDATE
  USING (id = ANY (app_tenant_ids())) WITH CHECK (id = ANY (app_tenant_ids()));
CREATE TRIGGER tenants_updated_at BEFORE UPDATE ON tenants
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Sites -----------------------------------------------------------------
CREATE TABLE sites (
  id         uuid NOT NULL,
  tenant_id  uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  name       text NOT NULL CHECK (length(name) BETWEEN 1 AND 200),
  timezone   text NOT NULL DEFAULT 'Europe/Berlin',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (id),
  UNIQUE (tenant_id, id),
  UNIQUE (tenant_id, name)
);
SELECT enable_tenant_rls('sites');
CREATE TRIGGER sites_updated_at BEFORE UPDATE ON sites
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Satellites --------------------------------------------------------------
CREATE TABLE satellites (
  id                     uuid NOT NULL,
  tenant_id              uuid NOT NULL,
  site_id                uuid NOT NULL,
  name                   text NOT NULL CHECK (length(name) BETWEEN 1 AND 200),
  status                 text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'active', 'revoked')),
  hostname               text NOT NULL DEFAULT '',
  agent_version          text NOT NULL DEFAULT '',
  protocol_version       integer NOT NULL DEFAULT 0,
  os                     text NOT NULL DEFAULT '',
  arch                   text NOT NULL DEFAULT '',
  last_seen_at           timestamptz,
  config_version_desired bigint NOT NULL DEFAULT 0,
  config_version_applied bigint NOT NULL DEFAULT 0,
  update_channel         text NOT NULL DEFAULT 'stable' CHECK (update_channel IN ('canary', 'stable')),
  created_at             timestamptz NOT NULL DEFAULT now(),
  updated_at             timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (id),
  UNIQUE (tenant_id, id),
  UNIQUE (tenant_id, name),
  FOREIGN KEY (tenant_id, site_id) REFERENCES sites (tenant_id, id) ON DELETE RESTRICT
);
SELECT enable_tenant_rls('satellites');
CREATE TRIGGER satellites_updated_at BEFORE UPDATE ON satellites
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE enrollment_tokens (
  id           uuid PRIMARY KEY,
  tenant_id    uuid NOT NULL,
  satellite_id uuid NOT NULL,
  token_hash   bytea NOT NULL UNIQUE,
  expires_at   timestamptz NOT NULL,
  used_at      timestamptz,
  created_at   timestamptz NOT NULL DEFAULT now(),
  FOREIGN KEY (tenant_id, satellite_id) REFERENCES satellites (tenant_id, id) ON DELETE CASCADE
);
SELECT enable_tenant_rls('enrollment_tokens');

CREATE TABLE satellite_certificates (
  serial             text PRIMARY KEY,
  tenant_id          uuid NOT NULL,
  satellite_id       uuid NOT NULL,
  fingerprint_sha256 bytea NOT NULL UNIQUE,
  not_before         timestamptz NOT NULL,
  not_after          timestamptz NOT NULL,
  revoked_at         timestamptz,
  created_at         timestamptz NOT NULL DEFAULT now(),
  FOREIGN KEY (tenant_id, satellite_id) REFERENCES satellites (tenant_id, id) ON DELETE CASCADE
);
SELECT enable_tenant_rls('satellite_certificates');
CREATE INDEX satellite_certificates_satellite_idx ON satellite_certificates (tenant_id, satellite_id);

-- Hosts -------------------------------------------------------------------
CREATE TABLE hosts (
  id            uuid NOT NULL,
  tenant_id     uuid NOT NULL,
  site_id       uuid NOT NULL,
  satellite_id  uuid NOT NULL,
  name          text NOT NULL CHECK (length(name) BETWEEN 1 AND 255),
  address       text NOT NULL CHECK (length(address) BETWEEN 1 AND 255),
  tags          text[] NOT NULL DEFAULT '{}',
  enabled       boolean NOT NULL DEFAULT true,
  host_check_id uuid,
  created_at    timestamptz NOT NULL DEFAULT now(),
  updated_at    timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (id),
  UNIQUE (tenant_id, id),
  UNIQUE (tenant_id, site_id, name),
  FOREIGN KEY (tenant_id, site_id) REFERENCES sites (tenant_id, id) ON DELETE RESTRICT,
  FOREIGN KEY (tenant_id, satellite_id) REFERENCES satellites (tenant_id, id) ON DELETE RESTRICT
);
SELECT enable_tenant_rls('hosts');
CREATE INDEX hosts_satellite_idx ON hosts (tenant_id, satellite_id);
CREATE TRIGGER hosts_updated_at BEFORE UPDATE ON hosts
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE host_dependencies (
  tenant_id      uuid NOT NULL,
  child_host_id  uuid NOT NULL,
  parent_host_id uuid NOT NULL,
  PRIMARY KEY (tenant_id, child_host_id, parent_host_id),
  CHECK (child_host_id <> parent_host_id),
  FOREIGN KEY (tenant_id, child_host_id) REFERENCES hosts (tenant_id, id) ON DELETE CASCADE,
  FOREIGN KEY (tenant_id, parent_host_id) REFERENCES hosts (tenant_id, id) ON DELETE CASCADE
);
SELECT enable_tenant_rls('host_dependencies');

-- Checks ------------------------------------------------------------------
CREATE TABLE checks (
  id               uuid NOT NULL,
  tenant_id        uuid NOT NULL,
  host_id          uuid NOT NULL,
  name             text NOT NULL CHECK (length(name) BETWEEN 1 AND 255),
  type             text NOT NULL CHECK (type IN ('ping', 'tcp', 'http', 'dns', 'snmp', 'self')),
  spec             jsonb NOT NULL DEFAULT '{}',
  thresholds       jsonb NOT NULL DEFAULT '[]',
  interval_s       integer NOT NULL DEFAULT 60 CHECK (interval_s BETWEEN 10 AND 86400),
  timeout_s        integer NOT NULL DEFAULT 10 CHECK (timeout_s BETWEEN 1 AND 300),
  retry_interval_s integer NOT NULL DEFAULT 30 CHECK (retry_interval_s BETWEEN 5 AND 86400),
  max_attempts     integer NOT NULL DEFAULT 3 CHECK (max_attempts BETWEEN 1 AND 10),
  enabled          boolean NOT NULL DEFAULT true,
  created_at       timestamptz NOT NULL DEFAULT now(),
  updated_at       timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (id),
  UNIQUE (tenant_id, id),
  UNIQUE (tenant_id, host_id, name),
  CHECK (timeout_s < interval_s),
  FOREIGN KEY (tenant_id, host_id) REFERENCES hosts (tenant_id, id) ON DELETE CASCADE
);
SELECT enable_tenant_rls('checks');
CREATE INDEX checks_host_idx ON checks (tenant_id, host_id);
CREATE TRIGGER checks_updated_at BEFORE UPDATE ON checks
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

ALTER TABLE hosts
  ADD CONSTRAINT hosts_host_check_fk FOREIGN KEY (tenant_id, host_check_id)
  REFERENCES checks (tenant_id, id) ON DELETE SET NULL (host_check_id);

-- Current state per check (updated on every result; HOT-update friendly).
CREATE TABLE check_state (
  check_id       uuid NOT NULL,
  tenant_id      uuid NOT NULL,
  soft_status    smallint NOT NULL DEFAULT 0,
  hard_status    smallint NOT NULL DEFAULT 0,
  state_type     text NOT NULL DEFAULT 'pending' CHECK (state_type IN ('pending', 'soft', 'hard')),
  attempt        integer NOT NULL DEFAULT 0,
  last_result_at timestamptz,
  last_change_at timestamptz,
  flap_ring      integer NOT NULL DEFAULT 0,
  is_flapping    boolean NOT NULL DEFAULT false,
  output         text NOT NULL DEFAULT '',
  updated_at     timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (check_id),
  FOREIGN KEY (tenant_id, check_id) REFERENCES checks (tenant_id, id) ON DELETE CASCADE
) WITH (fillfactor = 70);
SELECT enable_tenant_rls('check_state');

-- +goose Down
DROP TABLE check_state;
ALTER TABLE hosts DROP CONSTRAINT hosts_host_check_fk;
DROP TABLE checks;
DROP TABLE host_dependencies;
DROP TABLE hosts;
DROP TABLE satellite_certificates;
DROP TABLE enrollment_tokens;
DROP TABLE satellites;
DROP TABLE sites;
DROP TABLE tenants;
