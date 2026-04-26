package benchmark

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/Leumas-LSN/benchere/internal/proxmox"
)

// IPPool describes an inclusive range of IPv4 addresses workers can be
// assigned, plus the subnet mask (CIDR) and gateway used in cloud-init's
// ipconfig0.
type IPPool struct {
	From    net.IP
	To      net.IP
	CIDR    int
	Gateway net.IP
}

// ParseIPPool parses range "A.B.C.D-A.B.C.E" + cidr (1-32) + gateway "G.H.I.J".
func ParseIPPool(rangeStr string, cidr int, gateway string) (IPPool, error) {
	parts := strings.Split(rangeStr, "-")
	if len(parts) != 2 {
		return IPPool{}, fmt.Errorf("worker ip range %q: expected format A.B.C.D-A.B.C.E", rangeStr)
	}
	from := net.ParseIP(strings.TrimSpace(parts[0])).To4()
	to := net.ParseIP(strings.TrimSpace(parts[1])).To4()
	if from == nil || to == nil {
		return IPPool{}, fmt.Errorf("worker ip range %q: invalid IPv4 addresses", rangeStr)
	}
	if cidr < 1 || cidr > 32 {
		return IPPool{}, fmt.Errorf("worker cidr %d: must be 1-32", cidr)
	}
	gw := net.ParseIP(strings.TrimSpace(gateway)).To4()
	if gw == nil {
		return IPPool{}, fmt.Errorf("worker gateway %q: invalid IPv4 address", gateway)
	}
	if ipToInt(from) > ipToInt(to) {
		return IPPool{}, fmt.Errorf("worker ip range %s-%s: start > end", from, to)
	}
	return IPPool{From: from, To: to, CIDR: cidr, Gateway: gw}, nil
}

// IPs enumerates the addresses in the inclusive range.
func (p IPPool) IPs() []net.IP {
	var out []net.IP
	for cur := ipToInt(p.From); cur <= ipToInt(p.To); cur++ {
		out = append(out, intToIP(cur))
	}
	return out
}

// IPConfig returns the cloud-init ipconfig0 string for a given assigned IP.
// Format: "ip=10.90.0.200/24,gw=10.90.0.1".
func (p IPPool) IPConfig(ip net.IP) string {
	return fmt.Sprintf("ip=%s/%d,gw=%s", ip, p.CIDR, p.Gateway)
}

func ipToInt(ip net.IP) uint32 {
	ip = ip.To4()
	return uint32(ip[0])<<24 | uint32(ip[1])<<16 | uint32(ip[2])<<8 | uint32(ip[3])
}

func intToIP(n uint32) net.IP {
	return net.IPv4(byte(n>>24), byte(n>>16), byte(n>>8), byte(n))
}

// usedIPs returns the set of IPv4 addresses currently assigned to any VM on
// the given node, via Proxmox API. We parse each VM's ipconfig0 string and
// extract the ip= value. VMs using DHCP (ip=dhcp) are skipped because their
// runtime address can't be discovered without the guest agent.
func usedIPs(ctx context.Context, c *proxmox.Client, node string) (map[string]bool, error) {
	vmids, err := c.ListVMIDs(ctx, node)
	if err != nil {
		return nil, fmt.Errorf("list VMs on node %s: %w", node, err)
	}
	used := make(map[string]bool)
	for _, vmid := range vmids {
		cfg, err := c.GetVMIPConfig0(ctx, node, vmid)
		if err != nil || cfg == "" {
			continue
		}
		ip := extractIPFromIPConfig0(cfg)
		if ip != "" {
			used[ip] = true
		}
	}
	return used, nil
}

// extractIPFromIPConfig0 parses "ip=10.90.0.200/24,gw=10.90.0.1" and returns "10.90.0.200".
// Returns "" for "ip=dhcp" or unparseable values.
func extractIPFromIPConfig0(s string) string {
	for _, part := range strings.Split(s, ",") {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 || kv[0] != "ip" {
			continue
		}
		val := kv[1]
		if val == "dhcp" {
			return ""
		}
		// Strip /CIDR if present
		if i := strings.Index(val, "/"); i >= 0 {
			val = val[:i]
		}
		return val
	}
	return ""
}

// AllocateIPs picks N free IPs from the pool, excluding any that are
// currently assigned to other VMs on the node. Returns an error if the
// pool doesn't have enough free addresses.
func AllocateIPs(ctx context.Context, c *proxmox.Client, node string, pool IPPool, count int) ([]net.IP, error) {
	used, err := usedIPs(ctx, c, node)
	if err != nil {
		// Don't block on transient API issues — proceed and let CreateVM
		// surface a clearer error if we try to use a colliding IP.
		used = map[string]bool{}
	}
	var allocated []net.IP
	for _, candidate := range pool.IPs() {
		if used[candidate.String()] {
			continue
		}
		allocated = append(allocated, candidate)
		if len(allocated) == count {
			return allocated, nil
		}
	}
	return nil, fmt.Errorf("ip pool exhausted: need %d free IPs in %s-%s, have %d", count, pool.From, pool.To, len(allocated))
}
