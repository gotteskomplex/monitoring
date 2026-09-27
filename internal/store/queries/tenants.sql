-- name: CreateTenant :one
-- Platform operation: requires store.WithSystem (mon_app has no INSERT policy).
INSERT INTO tenants (id, slug, name)
VALUES (@id, @slug, @name)
RETURNING *;

-- name: GetTenant :one
SELECT * FROM tenants WHERE id = @id;

-- name: ListTenants :many
SELECT * FROM tenants ORDER BY name;

-- name: RenameTenant :execrows
UPDATE tenants SET name = @name WHERE id = @id;

-- name: DeleteTenant :execrows
-- Platform operation: requires store.WithSystem.
DELETE FROM tenants WHERE id = @id;
