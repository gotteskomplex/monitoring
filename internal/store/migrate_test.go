package store

import "testing"

func TestLatestMigration(t *testing.T) {
	v, err := LatestMigration()
	if err != nil {
		t.Fatal(err)
	}
	if v < 3 {
		t.Fatalf("latest migration = %d, want >= 3", v)
	}
}
