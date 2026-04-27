package proxmox_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Leumas-LSN/benchere/internal/proxmox"
)

func TestGetNodeStorages_ReturnsActiveStoragesOnly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api2/json/nodes/aqua/storage" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{"storage": "ceph", "type": "rbd", "content": "images,rootdir", "active": 1, "enabled": 1},
					{"storage": "local", "type": "dir", "content": "iso,backup", "active": 1, "enabled": 1},
					{"storage": "down-nfs", "type": "nfs", "content": "images", "active": 0, "enabled": 1},
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := proxmox.NewClient(srv.URL, "")
	got, err := c.GetNodeStorages(context.Background(), "aqua")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 active storages, got %d (%v)", len(got), got)
	}
	want := map[string]string{"ceph": "rbd", "local": "dir"}
	for _, s := range got {
		if want[s.ID] != s.Type {
			t.Errorf("unexpected storage %+v", s)
		}
	}
}
