-- Database bootstrap. Runs as superuser, idempotent.
-- Roles (see docs/ARCHITECTURE.md §6.2):
--   mon_migrator  owns the schema, runs migrations only
--   mon_app       application role, NO BYPASSRLS, not owner of any table
--   mon_system    BYPASSRLS, only for clearly scoped cross-tenant jobs
DO $$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'mon_migrator') THEN
    CREATE ROLE mon_migrator LOGIN;
  END IF;
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'mon_app') THEN
    CREATE ROLE mon_app LOGIN;
  END IF;
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'mon_system') THEN
    CREATE ROLE mon_system LOGIN;
  END IF;

  ALTER ROLE mon_migrator NOSUPERUSER NOCREATEROLE NOCREATEDB NOBYPASSRLS;
  ALTER ROLE mon_app      NOSUPERUSER NOCREATEROLE NOCREATEDB NOBYPASSRLS;
  ALTER ROLE mon_system   NOSUPERUSER NOCREATEROLE NOCREATEDB BYPASSRLS;

  EXECUTE format('ALTER ROLE mon_migrator PASSWORD %L', current_setting('mon.migrator_password'));
  EXECUTE format('ALTER ROLE mon_app PASSWORD %L', current_setting('mon.app_password'));
  EXECUTE format('ALTER ROLE mon_system PASSWORD %L', current_setting('mon.system_password'));

  EXECUTE format('ALTER DATABASE %I OWNER TO mon_migrator', current_database());
  EXECUTE format('REVOKE ALL ON DATABASE %I FROM PUBLIC', current_database());
  EXECUTE format('GRANT CONNECT ON DATABASE %I TO mon_app, mon_system', current_database());
END
$$;

CREATE EXTENSION IF NOT EXISTS timescaledb;

ALTER SCHEMA public OWNER TO mon_migrator;
REVOKE ALL ON SCHEMA public FROM PUBLIC;
GRANT USAGE ON SCHEMA public TO mon_app, mon_system;

ALTER DEFAULT PRIVILEGES FOR ROLE mon_migrator IN SCHEMA public
  GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO mon_app, mon_system;
ALTER DEFAULT PRIVILEGES FOR ROLE mon_migrator IN SCHEMA public
  GRANT USAGE, SELECT ON SEQUENCES TO mon_app, mon_system;
ALTER DEFAULT PRIVILEGES FOR ROLE mon_migrator IN SCHEMA public
  GRANT EXECUTE ON FUNCTIONS TO mon_app, mon_system;
