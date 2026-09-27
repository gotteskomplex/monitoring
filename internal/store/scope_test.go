package store

import (
	"testing"

	"github.com/google/uuid"
)

func TestTenantScope(t *testing.T) {
	a := uuid.MustParse("00000000-0000-0000-0000-00000000000a")
	b := uuid.MustParse("00000000-0000-0000-0000-00000000000b")

	tests := []struct {
		name    string
		in      []uuid.UUID
		setting string
		empty   bool
	}{
		{"none", nil, "{}", true},
		{"nil uuid only", []uuid.UUID{uuid.Nil}, "{}", true},
		{"single", []uuid.UUID{a}, "{" + a.String() + "}", false},
		{"dedupe and sort", []uuid.UUID{b, a, b, uuid.Nil}, "{" + a.String() + "," + b.String() + "}", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := TenantScope(tt.in...)
			if got := s.setting(); got != tt.setting {
				t.Errorf("setting = %q, want %q", got, tt.setting)
			}
			if s.Empty() != tt.empty {
				t.Errorf("Empty = %v, want %v", s.Empty(), tt.empty)
			}
		})
	}
}

func TestScopeContainsAndCopy(t *testing.T) {
	a, b := uuid.New(), uuid.New()
	s := TenantScope(a)
	if !s.Contains(a) || s.Contains(b) {
		t.Fatal("Contains wrong")
	}
	ids := s.TenantIDs()
	ids[0] = b
	if !s.Contains(a) {
		t.Fatal("TenantIDs must return a copy")
	}
}
