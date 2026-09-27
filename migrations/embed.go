// Package migrations embeds the versioned goose SQL migrations and the
// database bootstrap script (roles, extensions) that must run as superuser.
package migrations

import "embed"

// FS contains the numbered goose migrations (NNNNN_name.sql).
//
//go:embed *.sql
var FS embed.FS

// BootstrapSQL creates roles and extensions. It is idempotent and must be
// executed by a superuser before running migrations. It expects the custom
// settings mon.migrator_password, mon.app_password and mon.system_password.
//
//go:embed bootstrap/bootstrap.sql
var BootstrapSQL string
