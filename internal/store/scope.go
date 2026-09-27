// Package store is the only way to access the master database.
//
// Every request-path access goes through WithTenantScope, which opens a
// transaction and sets app.tenant_ids so that PostgreSQL row level security
// restricts all statements to the given tenants (ADR-0004). Cross-tenant
// platform jobs use WithSystem, which runs as the mon_system role.
package store

import (
	"slices"
	"strings"

	"github.com/google/uuid"
)

// Scope is the set of tenants a transaction may see and modify.
// The zero value is empty and is rejected by WithTenantScope.
type Scope struct {
	ids []uuid.UUID
}

// TenantScope builds a scope from tenant IDs. Nil UUIDs and duplicates are
// dropped; the result is sorted for deterministic settings.
func TenantScope(ids ...uuid.UUID) Scope {
	out := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if id == uuid.Nil || slices.Contains(out, id) {
			continue
		}
		out = append(out, id)
	}
	slices.SortFunc(out, func(a, b uuid.UUID) int { return strings.Compare(a.String(), b.String()) })
	return Scope{ids: out}
}

// TenantIDs returns a copy of the tenant IDs in the scope.
func (s Scope) TenantIDs() []uuid.UUID { return slices.Clone(s.ids) }

// Empty reports whether the scope contains no tenant.
func (s Scope) Empty() bool { return len(s.ids) == 0 }

// Contains reports whether id is part of the scope.
func (s Scope) Contains(id uuid.UUID) bool { return slices.Contains(s.ids, id) }

// setting renders the scope as a PostgreSQL uuid[] literal for app.tenant_ids.
func (s Scope) setting() string {
	parts := make([]string, len(s.ids))
	for i, id := range s.ids {
		parts[i] = id.String()
	}
	return "{" + strings.Join(parts, ",") + "}"
}
