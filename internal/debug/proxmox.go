package debug

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Leumas-LSN/benchere/internal/proxmox"
)

// ProxmoxCollector wraps a Proxmox client to gather cluster-level snapshots
// for the bundle. All methods are best-effort and return a stub describing
// the failure rather than aborting.
type ProxmoxCollector struct {
	Client *proxmox.Client
}

// NodesJSON returns the list of node names as pretty JSON.
func (p *ProxmoxCollector) NodesJSON(ctx context.Context) []byte {
	if p == nil || p.Client == nil {
		return []byte("# unavailable: no proxmox client\n")
	}
	nodes, err := p.Client.GetNodes(ctx)
	if err != nil {
		return []byte(fmt.Sprintf("# unavailable: %v\n", err))
	}
	data, _ := json.MarshalIndent(nodes, "", "  ")
	return append(data, '\n')
}

// NodeStatus returns the status JSON for one node, or a stub describing
// the failure.
func (p *ProxmoxCollector) NodeStatus(ctx context.Context, node string) []byte {
	if p == nil || p.Client == nil {
		return []byte("# unavailable: no proxmox client\n")
	}
	st, err := p.Client.GetNodeStatus(ctx, node)
	if err != nil {
		return []byte(fmt.Sprintf("# unavailable: %v\n", err))
	}
	data, _ := json.MarshalIndent(st, "", "  ")
	return append(data, '\n')
}

// StoragesJSON returns cluster-level storage configuration.
func (p *ProxmoxCollector) StoragesJSON(ctx context.Context) []byte {
	if p == nil || p.Client == nil {
		return []byte("# unavailable: no proxmox client\n")
	}
	storages, err := p.Client.GetStorages(ctx)
	if err != nil {
		return []byte(fmt.Sprintf("# unavailable: %v\n", err))
	}
	data, _ := json.MarshalIndent(storages, "", "  ")
	return append(data, '\n')
}

// PVEVersionJSON returns the Proxmox software version reported by one of
// the nodes. Tries the first node returned by GetNodes.
func (p *ProxmoxCollector) PVEVersionJSON(ctx context.Context) []byte {
	if p == nil || p.Client == nil {
		return []byte("# unavailable: no proxmox client\n")
	}
	nodes, err := p.Client.GetNodes(ctx)
	if err != nil || len(nodes) == 0 {
		return []byte(fmt.Sprintf("# unavailable: %v\n", err))
	}
	out, err := p.Client.RawGet(ctx, fmt.Sprintf("/nodes/%s/version", nodes[0]))
	if err != nil {
		return []byte(fmt.Sprintf("# unavailable: %v\n", err))
	}
	return out
}
