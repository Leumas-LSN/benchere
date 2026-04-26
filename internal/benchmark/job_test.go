package benchmark_test

import (
	"testing"

	"github.com/Leumas-LSN/benchere/internal/benchmark"
)

func TestCanTransition(t *testing.T) {
	cases := []struct {
		from, to string
		want     bool
	}{
		{"pending", "provisioning", true},
		{"pending", "running", false},
		{"provisioning", "running", true},
		{"provisioning", "failed", true},
		{"running", "done", true},
		{"done", "running", false},
		{"failed", "running", false},
	}
	for _, tc := range cases {
		got := benchmark.CanTransition(tc.from, tc.to)
		if got != tc.want {
			t.Errorf("CanTransition(%q, %q) = %v, want %v", tc.from, tc.to, got, tc.want)
		}
	}
}
