package debug

import (
	"context"
	"fmt"

	"github.com/Leumas-LSN/benchere/internal/proxmox"
)

// CephCollector queries the first node that exposes /ceph endpoints.
// All methods return a "# unavailable: <reason>" stub when Ceph is not
// reachable so the bundle stays uniform across clusters with and without
// Ceph.
type CephCollector struct {
	Client *proxmox.Client
}

// pickNode returns the first node from /nodes, or "" if no nodes are
// reachable.
func (c *CephCollector) pickNode(ctx context.Context) string {
	if c == nil || c.Client == nil {
		return ""
	}
	nodes, err := c.Client.GetNodes(ctx)
	if err != nil || len(nodes) == 0 {
		return ""
	}
	return nodes[0]
}

func (c *CephCollector) get(ctx context.Context, sub string) []byte {
	node := c.pickNode(ctx)
	if node == "" {
		return []byte("# unavailable: no proxmox node reachable\n")
	}
	out, err := c.Client.RawGet(ctx, fmt.Sprintf("/nodes/%s/ceph%s", node, sub))
	if err != nil {
		return []byte(fmt.Sprintf("# unavailable: %v\n", err))
	}
	return out
}

// Status returns /nodes/{n}/ceph/status as JSON.
func (c *CephCollector) Status(ctx context.Context) []byte { return c.get(ctx, "/status") }

// DF returns /nodes/{n}/ceph/df as JSON.
func (c *CephCollector) DF(ctx context.Context) []byte { return c.get(ctx, "/df") }

// Pools returns /nodes/{n}/ceph/pools as JSON.
func (c *CephCollector) Pools(ctx context.Context) []byte { return c.get(ctx, "/pools") }

// OSD returns /nodes/{n}/ceph/osd as JSON.
func (c *CephCollector) OSD(ctx context.Context) []byte { return c.get(ctx, "/osd") }

// Config returns the Ceph cluster config dump if exposed by the node, or a
// stub. The endpoint name has changed across PVE releases so we just try
// /config which is the long-standing path.
func (c *CephCollector) Config(ctx context.Context) []byte {
	return c.get(ctx, "/config")
}
