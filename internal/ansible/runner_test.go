package ansible_test

import (
	"strings"
	"testing"

	"github.com/Leumas-LSN/benchere/internal/ansible"
)

func TestGenerateInventory(t *testing.T) {
	r := ansible.NewRunner("/tmp/ansible-test", "/tmp/id_rsa", "/tmp/elbencho.deb")
	targets := []ansible.WorkerTarget{
		{IP: "10.97.97.101"},
		{IP: "10.97.97.102"},
	}
	inv, err := r.GenerateInventory(targets)
	if err != nil {
		t.Fatal(err)
	}
	if inv == "" {
		t.Fatal("empty inventory")
	}
	for _, target := range targets {
		if !strings.Contains(inv, target.IP) {
			t.Errorf("inventory missing IP %s", target.IP)
		}
	}
	if !strings.Contains(inv, "[workers]") {
		t.Error("inventory missing [workers] section")
	}
	if !strings.Contains(inv, "/tmp/id_rsa") {
		t.Error("inventory missing SSH key path")
	}
}

func TestNewRunner(t *testing.T) {
	r := ansible.NewRunner("/playbooks", "/key", "/deb")
	if r == nil {
		t.Fatal("expected non-nil runner")
	}
}
