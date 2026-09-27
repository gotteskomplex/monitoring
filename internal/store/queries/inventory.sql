-- name: CreateSite :one
INSERT INTO sites (id, tenant_id, name, timezone)
VALUES (@id, @tenant_id, @name, @timezone)
RETURNING *;

-- name: ListSites :many
SELECT * FROM sites ORDER BY name;

-- name: CreateSatellite :one
INSERT INTO satellites (id, tenant_id, site_id, name)
VALUES (@id, @tenant_id, @site_id, @name)
RETURNING *;

-- name: ListSatellites :many
SELECT * FROM satellites ORDER BY name;

-- name: CreateHost :one
INSERT INTO hosts (id, tenant_id, site_id, satellite_id, name, address, tags)
VALUES (@id, @tenant_id, @site_id, @satellite_id, @name, @address, @tags)
RETURNING *;

-- name: ListHosts :many
SELECT * FROM hosts ORDER BY name;

-- name: UpdateHostAddress :execrows
UPDATE hosts SET address = @address WHERE id = @id;

-- name: DeleteHost :execrows
DELETE FROM hosts WHERE id = @id;

-- name: CreateCheck :one
INSERT INTO checks (id, tenant_id, host_id, name, type, spec, interval_s, timeout_s)
VALUES (@id, @tenant_id, @host_id, @name, @type, @spec, @interval_s, @timeout_s)
RETURNING *;

-- name: ListChecks :many
SELECT * FROM checks ORDER BY name;
