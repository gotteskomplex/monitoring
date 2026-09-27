-- +goose Up
-- The goose bookkeeping table is created by goose before this migration and
-- inherits the default grants of schema public. Only mon_migrator may write it.
REVOKE INSERT, UPDATE, DELETE, TRUNCATE ON goose_db_version FROM mon_app, mon_system;

-- app_tenant_ids returns the tenant IDs of the current transaction scope.
-- It is set via SET LOCAL app.tenant_ids = '{uuid,...}' by store.WithTenantScope.
-- Missing or empty setting => empty array => no rows visible (fail closed).
-- +goose StatementBegin
CREATE FUNCTION app_tenant_ids() RETURNS uuid[]
LANGUAGE sql STABLE PARALLEL SAFE
AS $$
  SELECT COALESCE(NULLIF(current_setting('app.tenant_ids', true), '')::uuid[], '{}'::uuid[])
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION set_updated_at() RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  NEW.updated_at := now();
  RETURN NEW;
END
$$;
-- +goose StatementEnd

-- enable_tenant_rls applies the standard tenant isolation policy to a table
-- with a tenant_id column. Every tenant table MUST call it (meta-test enforces).
-- +goose StatementBegin
CREATE FUNCTION enable_tenant_rls(tbl regclass) RETURNS void
LANGUAGE plpgsql
AS $$
BEGIN
  EXECUTE format('ALTER TABLE %s ENABLE ROW LEVEL SECURITY', tbl);
  EXECUTE format('ALTER TABLE %s FORCE ROW LEVEL SECURITY', tbl);
  EXECUTE format(
    'CREATE POLICY tenant_isolation ON %s FOR ALL
       USING (tenant_id = ANY (app_tenant_ids()))
       WITH CHECK (tenant_id = ANY (app_tenant_ids()))', tbl);
END
$$;
-- +goose StatementEnd

-- +goose Down
DROP FUNCTION enable_tenant_rls(regclass);
DROP FUNCTION set_updated_at();
DROP FUNCTION app_tenant_ids();
