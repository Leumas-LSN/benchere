package proxmox_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Leumas-LSN/benchere/internal/proxmox"
)

func newTestServer(handler http.HandlerFunc) (*httptest.Server, *proxmox.Client) {
	srv := httptest.NewServer(handler)
	c := proxmox.NewClient(srv.URL, "user@pam!test=secret")
	return srv, c
}

func TestGetNodes(t *testing.T) {
	srv, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api2/json/nodes" {
			http.NotFound(w, r)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{"node": "pve-01", "status": "online"},
				{"node": "pve-02", "status": "online"},
			},
		})
	})
	defer srv.Close()

	nodes, err := c.GetNodes(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 2 {
		t.Errorf("want 2 nodes, got %d", len(nodes))
	}
	if nodes[0] != "pve-01" {
		t.Errorf("want pve-01, got %s", nodes[0])
	}
}

func TestGetNodeStatus(t *testing.T) {
	srv, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"cpu": 0.78,
				"memory": map[string]interface{}{
					"used":  4294967296.0,
					"total": 8589934592.0,
				},
				"loadavg": []float64{1.5, 1.2, 0.9},
			},
		})
	})
	defer srv.Close()

	status, err := c.GetNodeStatus(context.Background(), "pve-01")
	if err != nil {
		t.Fatal(err)
	}
	if status.CPUPct != 78.0 {
		t.Errorf("want CPUPct 78.0, got %f", status.CPUPct)
	}
	if status.RAMPct != 50.0 {
		t.Errorf("want RAMPct 50.0, got %f", status.RAMPct)
	}
	if status.LoadAvg != 1.5 {
		t.Errorf("want LoadAvg 1.5, got %f", status.LoadAvg)
	}
}

func TestCreateVM(t *testing.T) {
	srv, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/qemu") {
			json.NewEncoder(w).Encode(map[string]interface{}{"data": "UPID:pve-01:create"})
			return
		}
		if r.Method == "GET" && strings.Contains(r.URL.Path, "/tasks/") && strings.HasSuffix(r.URL.Path, "/status") {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": map[string]interface{}{
					"status":     "stopped",
					"exitstatus": "OK",
				},
			})
			return
		}
		http.NotFound(w, r)
	})
	defer srv.Close()

	vmid, err := c.CreateVM(context.Background(), proxmox.VMCreateParams{
		Node: "pve-01", VMID: 200, Name: "worker-1",
		Cores: 4, MemoryMB: 2048, StoragePool: "local-lvm",
		SSHKey: "ssh-rsa AAAA...", OSDiskGB: 16,
	})
	if err != nil {
		t.Fatal(err)
	}
	if vmid != 200 {
		t.Errorf("want vmid 200, got %d", vmid)
	}
}

func TestGetVMStatus(t *testing.T) {
	srv, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"status": "running",
				"cpu":    0.45,
			},
		})
	})
	defer srv.Close()

	status, err := c.GetVMStatus(context.Background(), "pve-01", 200)
	if err != nil {
		t.Fatal(err)
	}
	if status.Status != "running" {
		t.Errorf("want running, got %s", status.Status)
	}
	if status.CPUPct != 45.0 {
		t.Errorf("want CPUPct 45.0, got %f", status.CPUPct)
	}
}
