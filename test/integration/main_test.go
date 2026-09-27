//go:build integration

// Package integration contains tests that need real infrastructure
// (TimescaleDB via Testcontainers). Run with: make test-int
package integration

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/gotteskomplex/monitoring/internal/testutil/pgtest"
)

var env *pgtest.Env

func TestMain(m *testing.M) {
	ctx := context.Background()
	var err error
	env, err = pgtest.Start(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "start database:", err)
		os.Exit(1)
	}
	code := m.Run()
	_ = env.Terminate(ctx)
	os.Exit(code)
}
