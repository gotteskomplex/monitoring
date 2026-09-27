package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLookup(t *testing.T) {
	t.Setenv("MON_TEST_A", "plain")
	if v, _ := Lookup("MON_TEST_A", "def"); v != "plain" {
		t.Fatalf("got %q", v)
	}
	if v, _ := Lookup("MON_TEST_MISSING", "def"); v != "def" {
		t.Fatalf("got %q", v)
	}

	f := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(f, []byte("from-file\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MON_TEST_A_FILE", f)
	if v, _ := Lookup("MON_TEST_A", "def"); v != "from-file" {
		t.Fatalf("file must win, got %q", v)
	}

	t.Setenv("MON_TEST_B_FILE", filepath.Join(t.TempDir(), "nope"))
	if _, err := Lookup("MON_TEST_B", ""); err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestRequire(t *testing.T) {
	if _, err := Require("MON_TEST_EMPTY"); err == nil {
		t.Fatal("expected error")
	}
	t.Setenv("MON_TEST_SET", "x")
	if v, err := Require("MON_TEST_SET"); err != nil || v != "x" {
		t.Fatalf("got %q, %v", v, err)
	}
}
