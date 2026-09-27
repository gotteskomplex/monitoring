//go:build integration

package integration

import (
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gotteskomplex/monitoring/internal/testutil/pgtest"
)

func openSuperPool(t *testing.T) *pgxpool.Pool { return pgtest.Pool(t, env.SuperDSN) }
func openAppPool(t *testing.T) *pgxpool.Pool   { return pgtest.Pool(t, env.AppDSN) }
