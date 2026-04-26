package stress_test

import (
	"strings"
	"testing"

	"github.com/Leumas-LSN/benchere/internal/stress"
)

func TestNewRunner(t *testing.T) {
	r := stress.NewRunner("/tmp/id_rsa")
	if r == nil {
		t.Fatal("expected non-nil runner")
	}
}

func TestBuildCommand(t *testing.T) {
	cmd := stress.BuildCommand(stress.Config{
		Workers:   4,
		Timeout:   60,
		Stressors: []string{"cpu", "vm"},
	})
	if cmd == "" {
		t.Fatal("empty command")
	}
	for _, s := range []string{"stress-ng", "--cpu", "--vm", "--timeout"} {
		if !strings.Contains(cmd, s) {
			t.Errorf("command missing %q: %s", s, cmd)
		}
	}
	if !strings.Contains(cmd, "60s") {
		t.Errorf("timeout not in command: %s", cmd)
	}
	if !strings.Contains(cmd, "4") {
		t.Errorf("worker count not in command: %s", cmd)
	}
}
