package proxmox

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type VMCreateParams struct {
	Node        string
	VMID        int
	Name        string
	Cores       int
	MemoryMB    int
	OSDiskGB    int
	StoragePool string
	SSHKey      string
	ImagePath   string
	DataDisks   int
	DataDiskGB  int
	Bridge      string
	Cicustom    string
	IPConfig    string
}

type VMStatus struct {
	VMID   int
	Status string
	CPUPct float64
	IP     string
}

// readUPID drains the response body and returns the task UPID from the
// "data" envelope. Proxmox returns 200 immediately with this UPID; the
// task itself runs async and may still fail.
func readUPID(resp *http.Response) (string, error) {
	var env struct {
		Data string `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return "", err
	}
	if !strings.HasPrefix(env.Data, "UPID:") {
		return "", fmt.Errorf("unexpected proxmox response: %q", env.Data)
	}
	return env.Data, nil
}

func (c *Client) CreateVM(ctx context.Context, p VMCreateParams) (int, error) {
	form := url.Values{}
	form.Set("vmid", fmt.Sprintf("%d", p.VMID))
	form.Set("name", p.Name)
	form.Set("cores", fmt.Sprintf("%d", p.Cores))
	form.Set("memory", fmt.Sprintf("%d", p.MemoryMB))
	bridge := p.Bridge
	if bridge == "" {
		bridge = "vmbr0"
	}
	form.Set("net0", "virtio,bridge="+bridge)
	form.Set("ide2", p.StoragePool+":cloudinit")
	ipConfig := p.IPConfig
	if ipConfig == "" {
		ipConfig = "ip=dhcp"
	}
	form.Set("ipconfig0", ipConfig)
	form.Set("ciuser", "root")
	encodedKey := strings.ReplaceAll(url.QueryEscape(strings.TrimSpace(p.SSHKey)), "+", "%20")
	form.Set("sshkeys", encodedKey)
	if p.ImagePath != "" {
		form.Set("scsi0", fmt.Sprintf("%s:0,import-from=%s,size=%dG", p.StoragePool, p.ImagePath, p.OSDiskGB))
	}
	form.Set("agent", "enabled=1,fstrim_cloned_disks=1")
	if p.Cicustom != "" {
		form.Set("cicustom", "user="+p.Cicustom)
	}
	form.Set("scsihw", "virtio-scsi-pci")
	form.Set("boot", "order=scsi0")
	for i := 0; i < p.DataDisks; i++ {
		form.Set(fmt.Sprintf("scsi%d", i+1), fmt.Sprintf("%s:%d", p.StoragePool, p.DataDiskGB))
	}
	resp, err := c.do(ctx, "POST", fmt.Sprintf("/nodes/%s/qemu", p.Node), strings.NewReader(form.Encode()))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		msg := strings.TrimSpace(string(b))
		if msg == "" {
			msg = resp.Status
		}
		return 0, fmt.Errorf("create vm failed: %s", msg)
	}
	upid, err := readUPID(resp)
	if err != nil {
		return 0, fmt.Errorf("create vm %d: %w", p.VMID, err)
	}
	if err := c.waitForTask(ctx, p.Node, upid); err != nil {
		return 0, fmt.Errorf("create vm %d: %w", p.VMID, err)
	}
	return p.VMID, nil
}

func (c *Client) StartVM(ctx context.Context, node string, vmid int) error {
	resp, err := c.do(ctx, "POST", fmt.Sprintf("/nodes/%s/qemu/%d/status/start", node, vmid), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("start vm %d failed: %s", vmid, resp.Status)
	}
	upid, err := readUPID(resp)
	if err != nil {
		return fmt.Errorf("start vm %d: %w", vmid, err)
	}
	if err := c.waitForTask(ctx, node, upid); err != nil {
		return fmt.Errorf("start vm %d: %w", vmid, err)
	}
	return nil
}

func (c *Client) DeleteVM(ctx context.Context, node string, vmid int) error {
	resp, err := c.do(ctx, "DELETE", fmt.Sprintf("/nodes/%s/qemu/%d?purge=1&destroy-unreferenced-disks=1", node, vmid), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil
	}
	upid, err := readUPID(resp)
	if err != nil {
		return nil
	}
	_ = c.waitForTask(ctx, node, upid)
	return nil
}

func (c *Client) GetVMStatus(ctx context.Context, node string, vmid int) (VMStatus, error) {
	var raw struct {
		Status string  `json:"status"`
		CPU    float64 `json:"cpu"`
	}
	if err := c.getJSON(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/status/current", node, vmid), &raw); err != nil {
		return VMStatus{}, err
	}
	return VMStatus{VMID: vmid, Status: raw.Status, CPUPct: raw.CPU * 100}, nil
}

// WaitForIP polls the QEMU guest agent until an IPv4 address is reported.
// Requires qemu-guest-agent running inside the VM. Bounded by a 5-minute
// deadline so a stuck/missing agent fails the job loudly instead of hanging.
func (c *Client) WaitForIP(ctx context.Context, node string, vmid int) (string, error) {
	deadline := time.Now().Add(5 * time.Minute)
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}
		var raw struct {
			Result []struct {
				Name        string `json:"name"`
				IPAddresses []struct {
					IPAddress     string `json:"ip-address"`
					IPAddressType string `json:"ip-address-type"`
				} `json:"ip-addresses"`
			} `json:"result"`
		}
		err := c.getJSON(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/agent/network-get-interfaces", node, vmid), &raw)
		if err == nil {
			for _, iface := range raw.Result {
				if iface.Name == "lo" {
					continue
				}
				for _, addr := range iface.IPAddresses {
					if addr.IPAddressType == "ipv4" {
						return addr.IPAddress, nil
					}
				}
			}
		}
		if time.Now().After(deadline) {
			return "", fmt.Errorf("waiting for IP timed out for vmid %d (qemu-guest-agent not responding after 5m)", vmid)
		}
		time.Sleep(3 * time.Second)
	}
}

// NextVMID returns the next available VMID from the Proxmox cluster.
func (c *Client) NextVMID(ctx context.Context) (int, error) {
	var id int
	if err := c.getJSON(ctx, "/cluster/nextid", &id); err != nil {
		// Fall back to timestamp-based allocation if cluster API unavailable
		return 200 + int(time.Now().UnixNano()%8000), nil
	}
	return id, nil
}


// StopVM sends a power-off to the VM and waits for it to stop.
func (c *Client) StopVM(ctx context.Context, node string, vmid int) error {
	resp, err := c.do(ctx, "POST", fmt.Sprintf("/nodes/%s/qemu/%d/status/stop", node, vmid), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("stop vm %d: %s", vmid, resp.Status)
	}
	if upid, err := readUPID(resp); err == nil {
		_ = c.waitForTask(ctx, node, upid)
	}
	for i := 0; i < 20; i++ {
		time.Sleep(2 * time.Second)
		s, err := c.GetVMStatus(ctx, node, vmid)
		if err != nil || s.Status == "stopped" {
			return nil
		}
	}
	return nil
}

// InjectSSHKey writes pubkey to /root/.ssh/authorized_keys via the QEMU guest agent.
// The key is base64-encoded to avoid any shell escaping issues.
func (c *Client) InjectSSHKey(ctx context.Context, node string, vmid int, pubkey string) error {
	b64 := base64.StdEncoding.EncodeToString([]byte(strings.TrimSpace(pubkey) + "\n"))
	script := fmt.Sprintf(
		"mkdir -p /root/.ssh && printf '%%s' %s | base64 -d > /root/.ssh/authorized_keys && chmod 700 /root/.ssh && chmod 600 /root/.ssh/authorized_keys",
		b64,
	)
	resp, err := c.doJSON(ctx, "POST", fmt.Sprintf("/nodes/%s/qemu/%d/agent/exec", node, vmid), map[string]interface{}{
		"command": []string{"bash", "-c", script},
	})
	if err != nil {
		return fmt.Errorf("agent exec: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("agent exec: %s", strings.TrimSpace(string(b)))
	}
	var env struct {
		Data struct {
			PID int `json:"pid"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return fmt.Errorf("agent exec decode: %w", err)
	}
	pid := env.Data.PID

	for i := 0; i < 30; i++ {
		time.Sleep(2 * time.Second)
		var status struct {
			Exited   int `json:"exited"`
			Exitcode int `json:"exitcode"`
		}
		if err := c.getJSON(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/agent/exec-status?pid=%d", node, vmid, pid), &status); err != nil {
			continue
		}
		if status.Exited == 1 {
			if status.Exitcode != 0 {
				return fmt.Errorf("ssh key injection exit code %d", status.Exitcode)
			}
			return nil
		}
	}
	return fmt.Errorf("ssh key injection timed out")
}

// ListVMIDs returns all VM IDs declared on a node, regardless of state.
// Used by the IP allocator to find which addresses are already in use
// before assigning a static IP to a new worker.
func (c *Client) ListVMIDs(ctx context.Context, node string) ([]int, error) {
	var raw []struct {
		VMID int `json:"vmid"`
	}
	if err := c.getJSON(ctx, fmt.Sprintf("/nodes/%s/qemu", node), &raw); err != nil {
		return nil, err
	}
	ids := make([]int, 0, len(raw))
	for _, r := range raw {
		ids = append(ids, r.VMID)
	}
	return ids, nil
}

// GetVMIPConfig0 returns the cloud-init ipconfig0 string of a VM, e.g.
// "ip=10.90.0.200/24,gw=10.90.0.1" or "ip=dhcp". Returns "" when unset.
func (c *Client) GetVMIPConfig0(ctx context.Context, node string, vmid int) (string, error) {
	var raw struct {
		IPConfig0 string `json:"ipconfig0"`
	}
	if err := c.getJSON(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/config", node, vmid), &raw); err != nil {
		return "", err
	}
	return raw.IPConfig0, nil
}

// WaitForSSH polls TCP 22 on the given IP until a connection succeeds, or
// the 5-minute deadline is reached. Used after assigning a static IP via
// cloud-init: we already know the worker's IP so we don't need the QEMU
// guest agent to discover it. As soon as sshd accepts a TCP connection the
// VM is ready for Ansible.
func (c *Client) WaitForSSH(ctx context.Context, ip string) error {
	deadline := time.Now().Add(5 * time.Minute)
	dialer := net.Dialer{Timeout: 3 * time.Second}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(ip, "22"))
		if err == nil {
			conn.Close()
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("ssh on %s not reachable after 5m", ip)
		}
		time.Sleep(3 * time.Second)
	}
}
