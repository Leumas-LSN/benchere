package proxmox

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"
)

// EnsureCloudImage downloads a cloud image to Proxmox storage if not already present.
// imageStorage must support 'import' content (e.g. local dir-type storage).
// Returns the volume ID (e.g. "local:import/debian-12-genericcloud-amd64.qcow2").
func (c *Client) EnsureCloudImage(ctx context.Context, node, storage, imageURL string) (string, error) {
	filename := imageURL[strings.LastIndex(imageURL, "/")+1:]
	expectedVolid := storage + ":import/" + filename

	// Check if already present
	var vols []struct {
		Volid string `json:"volid"`
	}
	_ = c.getJSON(ctx, fmt.Sprintf("/nodes/%s/storage/%s/content", node, storage), &vols)
	for _, v := range vols {
		if v.Volid == expectedVolid {
			return v.Volid, nil
		}
	}

	form := url.Values{}
	form.Set("url", imageURL)
	form.Set("filename", filename)
	form.Set("content", "import")
	form.Set("verify-certificates", "0")
	resp, err := c.do(ctx, "POST", fmt.Sprintf("/nodes/%s/storage/%s/download-url", node, storage), strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("download cloud image: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		msg := parseProxmoxError(b)
		if msg == "" {
			msg = resp.Status
		}
		if strings.Contains(msg, "is not configured for content-type 'import'") {
			return "", fmt.Errorf("storage %q does not allow cloud-image imports. Fix: Datacenter → Storage → %s → Content (check 'Import'), or via CLI: pvesm set %s --content iso,vztmpl,backup,snippets,import,rootdir", storage, storage, storage)
		}
		return "", fmt.Errorf("download cloud image: %s", msg)
	}

	var envelope struct {
		Data string `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return "", fmt.Errorf("parse download task: %w", err)
	}

	if err := c.waitForTask(ctx, node, envelope.Data); err != nil {
		return "", fmt.Errorf("cloud image download failed: %w", err)
	}

	return expectedVolid, nil
}

// waitForTask polls a Proxmox task UPID until it stops, returning an error
// if exitstatus != "OK" or the 10-minute deadline is reached. A nil error
// means the task completed successfully.
func (c *Client) waitForTask(ctx context.Context, node, upid string) error {
	deadline := time.Now().Add(10 * time.Minute)
	path := fmt.Sprintf("/nodes/%s/tasks/%s/status", node, url.PathEscape(upid))
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		var status struct {
			Status     string `json:"status"`
			Exitstatus string `json:"exitstatus"`
		}
		if err := c.getJSON(ctx, path, &status); err == nil {
			if status.Status == "stopped" {
				if status.Exitstatus != "OK" {
					return fmt.Errorf("task exited with: %s", status.Exitstatus)
				}
				return nil
			}
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("task %s did not finish within 10m", upid)
		}
		time.Sleep(1 * time.Second)
	}
}

type StorageInfo struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Content string `json:"content"`
}

func (c *Client) GetStorages(ctx context.Context) ([]StorageInfo, error) {
	var raw []struct {
		Storage string `json:"storage"`
		Type    string `json:"type"`
		Content string `json:"content"`
	}
	if err := c.getJSON(ctx, "/storage", &raw); err != nil {
		return nil, err
	}
	result := make([]StorageInfo, len(raw))
	for i, s := range raw {
		result[i] = StorageInfo{ID: s.Storage, Type: s.Type, Content: s.Content}
	}
	return result, nil
}


// parseProxmoxError extracts the human-readable message from a Proxmox API error
// response body, which is JSON of the form {"data":null,"message":"..."}.
// Falls back to the raw trimmed body if no message field is present.
func parseProxmoxError(body []byte) string {
	var env struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &env); err == nil && env.Message != "" {
		return strings.TrimSpace(env.Message)
	}
	return strings.TrimSpace(string(body))
}
