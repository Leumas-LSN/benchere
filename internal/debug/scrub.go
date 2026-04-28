// Package debug builds tar.gz support bundles for a finished job.
//
// A bundle contains a SQLite snapshot of the live DB, the JobConfig used by
// the orchestrator, scrubbed settings, journalctl logs, raw elbencho /
// ansible output, per-worker sysinfo captured before cleanup, and a best
// effort dump of the Proxmox cluster + Ceph state. Secrets are scrubbed
// from settings before they enter the archive. The bundle is streamed to
// disk or HTTP, never buffered in memory.
package debug

import (
	"encoding/json"
	"strings"
)

const scrubMarker = "***SCRUBBED***"

// secretSubstrings is the lowercase substring set used to decide whether a
// settings/config key looks like a secret. Anything matching becomes
// scrubMarker before it ends up in the bundle.
var secretSubstrings = []string{
	"password",
	"passwd",
	"secret",
	"token",
	"apikey",
	"api_key",
	"private_key",
	"privatekey",
	"ssh_key",
}

// looksSecret reports whether a key (case-insensitive) matches one of the
// known secret patterns.
func looksSecret(key string) bool {
	lk := strings.ToLower(key)
	for _, s := range secretSubstrings {
		if strings.Contains(lk, s) {
			return true
		}
	}
	return false
}

// ScrubSettingsMap applies looksSecret to a flat map and replaces the value
// of any matching key with scrubMarker. Returns a new map; the input is not
// mutated.
func ScrubSettingsMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		if looksSecret(k) {
			out[k] = scrubMarker
		} else {
			out[k] = v
		}
	}
	return out
}

// ScrubSettingsJSON pretty-prints the input map as JSON with keys sorted,
// after scrubbing secrets.
func ScrubSettingsJSON(in map[string]string) ([]byte, error) {
	clean := ScrubSettingsMap(in)
	return json.MarshalIndent(clean, "", "  ")
}

// ScrubStorageCfgText replaces the value of any line matching a known secret
// keyword with scrubMarker. Operates per line so it can be applied to free
// form Proxmox storage.cfg / etc files captured as text.
func ScrubStorageCfgText(text string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "#") {
			continue
		}
		// Proxmox-style: "  password foo" or "  shared-secret bar"
		fields := strings.Fields(trim)
		if len(fields) >= 2 && looksSecret(fields[0]) {
			indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
			lines[i] = indent + fields[0] + " " + scrubMarker
			continue
		}
		// Generic "key=value": find first = and check the key.
		if eq := strings.IndexByte(line, '='); eq > 0 {
			key := strings.TrimSpace(line[:eq])
			if looksSecret(key) {
				lines[i] = line[:eq+1] + scrubMarker
			}
		}
	}
	return strings.Join(lines, "\n")
}
