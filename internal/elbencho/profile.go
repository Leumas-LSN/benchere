package elbencho

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ProfileToConfig returns an elbencho config file content from a profile's stored config.
// If the stored content is already in elbencho KEY=VALUE format, it is returned as-is.
// If it is JSON (legacy format), it is converted.
func ProfileToConfig(configJSON string) (string, error) {
	trimmed := strings.TrimSpace(configJSON)
	if strings.HasPrefix(trimmed, "#") || (strings.Contains(trimmed, "=") && !strings.HasPrefix(trimmed, "{")) {
		return trimmed + "\n", nil
	}
	return profileJSONToConfig(configJSON)
}

type profileSpec struct {
	BlockSize string `json:"block_size"`
	ReadPct   int    `json:"read_pct"`
	RandomPct int    `json:"random_pct"`
}

func profileJSONToConfig(configJSON string) (string, error) {
	var p profileSpec
	if err := json.Unmarshal([]byte(configJSON), &p); err != nil {
		return "", fmt.Errorf("parse profile: %w", err)
	}

	var lines []string
	lines = append(lines, "block="+p.BlockSize)

	switch {
	case p.ReadPct == 100:
		lines = append(lines, "read=true")
	case p.ReadPct == 0:
		lines = append(lines, "write=true")
	default:
		lines = append(lines, "write=true")
		lines = append(lines, fmt.Sprintf("rwmixpct=%d", p.ReadPct))
	}

	if p.RandomPct > 0 {
		lines = append(lines, "rand=true")
	}
	lines = append(lines, "threads=8")
	lines = append(lines, "iterations=1")

	return strings.Join(lines, "\n") + "\n", nil
}
