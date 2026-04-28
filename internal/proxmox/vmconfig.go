package proxmox

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

// RawGet returns the pretty-printed "data" field of a Proxmox API GET as
// JSON bytes. Used by the debug bundle to capture endpoints we do not
// model with a typed client method (Ceph status, /version, ...). Returns
// the body untouched as JSON if the response is not a JSON envelope.
func (c *Client) RawGet(ctx context.Context, path string) ([]byte, error) {
	resp, err := c.do(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			msg = resp.Status
		}
		return nil, fmt.Errorf("proxmox %s: %s", path, msg)
	}
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return body, nil
	}
	if len(envelope.Data) == 0 {
		return body, nil
	}
	var pretty interface{}
	if err := json.Unmarshal(envelope.Data, &pretty); err != nil {
		return envelope.Data, nil
	}
	out, err := json.MarshalIndent(pretty, "", "  ")
	if err != nil {
		return envelope.Data, nil
	}
	return append(out, '\n'), nil
}

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
