package proxmox

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// GetVMConfig returns the raw configuration of a VM as a sorted "key: value"
// text block. This is the JSON of /nodes/{node}/qemu/{vmid}/config rendered
// in stable order so it can be diffed across runs and is human readable.
// Used by the debug bundle to verify scsi*,cache=none and aio=native are
// applied as expected.
func (c *Client) GetVMConfig(ctx context.Context, node string, vmid int) (string, error) {
	var raw map[string]interface{}
	if err := c.getJSON(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/config", node, vmid), &raw); err != nil {
		return "", err
	}
	keys := make([]string, 0, len(raw))
	for k := range raw {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		v := raw[k]
		switch vv := v.(type) {
		case string:
			fmt.Fprintf(&b, "%s: %s\n", k, vv)
		default:
			data, err := json.Marshal(v)
			if err != nil {
				fmt.Fprintf(&b, "%s: <error: %v>\n", k, err)
				continue
			}
			fmt.Fprintf(&b, "%s: %s\n", k, string(data))
		}
	}
	return b.String(), nil
}
