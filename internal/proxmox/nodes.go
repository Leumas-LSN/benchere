package proxmox

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

type NodeStatus struct {
	Node    string
	CPUPct  float64
	RAMPct  float64
	LoadAvg float64
}

func (c *Client) GetNodes(ctx context.Context) ([]string, error) {
	var raw []struct {
		Node string `json:"node"`
	}
	if err := c.getJSON(ctx, "/nodes", &raw); err != nil {
		return nil, err
	}
	nodes := make([]string, len(raw))
	for i, n := range raw {
		nodes[i] = n.Node
	}
	return nodes, nil
}

func (c *Client) GetNodeStatus(ctx context.Context, node string) (NodeStatus, error) {
	var raw struct {
		CPU    float64 `json:"cpu"`
		Memory struct {
			Used  float64 `json:"used"`
			Total float64 `json:"total"`
		} `json:"memory"`
		LoadAvg []string `json:"loadavg"`
	}
	if err := c.getJSON(ctx, fmt.Sprintf("/nodes/%s/status", node), &raw); err != nil {
		return NodeStatus{}, err
	}
	ramPct := 0.0
	if raw.Memory.Total > 0 {
		ramPct = raw.Memory.Used / raw.Memory.Total * 100
	}
	loadAvg := 0.0
	if len(raw.LoadAvg) > 0 {
		loadAvg, _ = strconv.ParseFloat(raw.LoadAvg[0], 64)
	}
	return NodeStatus{
		Node:    node,
		CPUPct:  raw.CPU * 100,
		RAMPct:  ramPct,
		LoadAvg: loadAvg,
	}, nil
}


// Bridge describes a Proxmox network bridge usable for worker VMs.
type Bridge struct {
	Name    string
	Type    string
	Active  bool
	Address string
	Ports   string
	Comments string
}

// GetBridges returns all bridges (Linux + OVS) declared on the given node.
// type=any_bridge filters out physical NICs and aliases. Bridges with active=0
// are still returned so the user can see them in the UI and pick one to apply
// once it's brought up.
func (c *Client) GetBridges(ctx context.Context, node string) ([]Bridge, error) {
	var raw []struct {
		Iface       string `json:"iface"`
		Type        string `json:"type"`
		Active      int    `json:"active"`
		Address     string `json:"address"`
		BridgePorts string `json:"bridge_ports"`
		Comments    string `json:"comments"`
	}
	if err := c.getJSON(ctx, fmt.Sprintf("/nodes/%s/network?type=any_bridge", node), &raw); err != nil {
		return nil, err
	}
	out := make([]Bridge, 0, len(raw))
	for _, r := range raw {
		out = append(out, Bridge{
			Name:     r.Iface,
			Type:     r.Type,
			Active:   r.Active == 1,
			Address:  r.Address,
			Ports:    r.BridgePorts,
			Comments: strings.TrimSpace(r.Comments),
		})
	}
	return out, nil
}
