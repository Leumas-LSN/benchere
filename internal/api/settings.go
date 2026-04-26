package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/Leumas-LSN/benchere/internal/proxmox"
)

type settingsPayload struct {
	ProxmoxURL         string `json:"proxmox_url"`
	ProxmoxTokenID     string `json:"proxmox_token_id"`
	ProxmoxTokenSecret string `json:"proxmox_token_secret"`
	StoragePool        string `json:"storage_pool"`
	ImageStorage       string `json:"image_storage"`
	ProxmoxNode        string `json:"proxmox_node"`
	NetworkBridge  string `json:"network_bridge"`
	WorkerIPPool   string `json:"worker_ip_pool"`
	WorkerCIDR     string `json:"worker_cidr"`
	WorkerGateway  string `json:"worker_gateway"`
	SSHKeyPath     string `json:"ssh_key_path"`
}

func splitToken(t string) (id, secret string) {
	if i := strings.Index(t, "="); i >= 0 {
		return t[:i], t[i+1:]
	}
	return t, ""
}

func (s *Server) getSettings(w http.ResponseWriter, r *http.Request) {
	keys := []string{"proxmox_url", "proxmox_token", "storage_pool", "image_storage", "proxmox_node", "network_bridge", "worker_ip_pool", "worker_cidr", "worker_gateway", "ssh_key_path"}
	m := make(map[string]string)
	for _, k := range keys {
		v, _ := s.DB.GetSetting(k)
		m[k] = v
	}
	tokenID, tokenSecret := splitToken(m["proxmox_token"])
	delete(m, "proxmox_token")
	m["proxmox_token_id"] = tokenID
	m["proxmox_token_secret"] = tokenSecret

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(m)
}

func (s *Server) postSettings(w http.ResponseWriter, r *http.Request) {
	var p settingsPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	token := ""
	if p.ProxmoxTokenID != "" && p.ProxmoxTokenSecret != "" {
		token = p.ProxmoxTokenID + "=" + p.ProxmoxTokenSecret
	} else if p.ProxmoxTokenID != "" {
		token = p.ProxmoxTokenID
	}

	// Validate image_storage has the 'import' content type and that the network
	// bridge actually exists on the chosen node. Both checks are skipped if the
	// caller hasn't set proxmox_url/token yet (initial save), so first-time
	// configuration isn't blocked by the validators themselves.
	if p.ProxmoxURL != "" && token != "" {
		client := proxmox.NewClient(p.ProxmoxURL, token)

		if p.ImageStorage != "" {
			if storages, err := client.GetStorages(r.Context()); err == nil {
				if msg := validateImportStorage(storages, p.ImageStorage); msg != "" {
					http.Error(w, msg, http.StatusBadRequest)
					return
				}
			}
		}

		if p.NetworkBridge != "" && p.ProxmoxNode != "" {
			if bridges, err := client.GetBridges(r.Context(), p.ProxmoxNode); err == nil {
				if msg := validateBridge(bridges, p.NetworkBridge); msg != "" {
					http.Error(w, msg, http.StatusBadRequest)
					return
				}
			}
		}
	}

	pairs := map[string]string{
		"proxmox_url":     p.ProxmoxURL,
		"proxmox_node":    p.ProxmoxNode,
		"storage_pool":    p.StoragePool,
		"image_storage":   p.ImageStorage,
		"network_bridge":  p.NetworkBridge,
		"worker_ip_pool":  p.WorkerIPPool,
		"worker_cidr":     p.WorkerCIDR,
		"worker_gateway":  p.WorkerGateway,
		"ssh_key_path":    p.SSHKeyPath,
	}
	if token != "" {
		pairs["proxmox_token"] = token
	}
	for k, v := range pairs {
		if v != "" {
			if err := s.DB.SetSetting(k, v); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func validateImportStorage(storages []proxmox.StorageInfo, name string) string {
	for _, st := range storages {
		if st.ID != name {
			continue
		}
		for _, c := range strings.Split(st.Content, ",") {
			if strings.TrimSpace(c) == "import" {
				return ""
			}
		}
		return fmt.Sprintf("storage %q does not allow cloud-image imports. Enable 'Import' in Datacenter → Storage → %s → Content, or run on a Proxmox node: pvesm set %s --content iso,vztmpl,backup,snippets,import,rootdir", name, name, name)
	}
	return fmt.Sprintf("storage %q not found on this Proxmox cluster.", name)
}

func validateBridge(bridges []proxmox.Bridge, name string) string {
	available := make([]string, 0, len(bridges))
	for _, b := range bridges {
		available = append(available, b.Name)
		if b.Name == name {
			if !b.Active {
				return fmt.Sprintf("bridge %q exists but is not active on the node. Bring it up first (ifup %s) or pick another bridge.", name, name)
			}
			return ""
		}
	}
	return fmt.Sprintf("bridge %q not found on this node. Available: %s", name, strings.Join(available, ", "))
}

func (s *Server) testProxmox(w http.ResponseWriter, r *http.Request) {
	proxmoxURL, err := s.DB.GetSetting("proxmox_url")
	if err != nil {
		http.Error(w, "failed to read settings: "+err.Error(), http.StatusInternalServerError)
		return
	}
	proxmoxToken, err := s.DB.GetSetting("proxmox_token")
	if err != nil {
		http.Error(w, "failed to read settings: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if proxmoxURL == "" {
		http.Error(w, "proxmox_url not configured", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	client := proxmox.NewClient(proxmoxURL, proxmoxToken)
	nodes, err := client.GetNodes(r.Context())
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "nodes": nodes})
}

func (s *Server) getStorages(w http.ResponseWriter, r *http.Request) {
	proxmoxURL, _ := s.DB.GetSetting("proxmox_url")
	proxmoxToken, _ := s.DB.GetSetting("proxmox_token")
	if proxmoxURL == "" {
		http.Error(w, "proxmox_url not configured", http.StatusBadRequest)
		return
	}
	client := proxmox.NewClient(proxmoxURL, proxmoxToken)
	storages, err := client.GetStorages(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(storages)
}

func (s *Server) getBridges(w http.ResponseWriter, r *http.Request) {
	proxmoxURL, _ := s.DB.GetSetting("proxmox_url")
	proxmoxToken, _ := s.DB.GetSetting("proxmox_token")
	node, _ := s.DB.GetSetting("proxmox_node")
	if node == "" {
		node = r.URL.Query().Get("node")
	}
	if proxmoxURL == "" {
		http.Error(w, "proxmox_url not configured", http.StatusBadRequest)
		return
	}
	if node == "" {
		http.Error(w, "proxmox_node not configured (set it first or pass ?node=<name>)", http.StatusBadRequest)
		return
	}
	client := proxmox.NewClient(proxmoxURL, proxmoxToken)
	bridges, err := client.GetBridges(r.Context(), node)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bridges)
}
