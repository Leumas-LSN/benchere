package report

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"

	"github.com/Leumas-LSN/benchere/internal/db"
)

// InfraLine summarises the tested infrastructure as a one-line dense string.
type InfraLine struct {
	WorkerCount int
	WorkerCPU   int
	WorkerRAMMB int
	DataDiskGB  int
	StoragePool string
	Nodes       []string
}

// computeInfraLine derives infra summary from job + workers list.
// db.Job in the current schema does not carry worker hardware config (CPU,
// RAM, disk, storage pool) so those fields remain zero / empty here. The
// template hides them when zero.
func computeInfraLine(job db.Job, workers []db.Worker) InfraLine {
	_ = job // kept for signature stability and future extension
	nodeSet := map[string]bool{}
	for _, w := range workers {
		if w.ProxmoxNode != "" {
			nodeSet[w.ProxmoxNode] = true
		}
	}
	nodes := make([]string, 0, len(nodeSet))
	for n := range nodeSet {
		nodes = append(nodes, n)
	}
	sort.Strings(nodes)
	return InfraLine{
		WorkerCount: len(workers),
		Nodes:       nodes,
	}
}

// ProfileConfig is the typed view of a profile used for the report.
type ProfileConfig struct {
	Name               string
	BlockSize          string
	RWRatio            string
	Pattern            string
	RuntimeSec         int
	ThresholdIOPSR     int
	ThresholdIOPSW     int
	ThresholdLatencyMs float64
}

// profileFetcher is the minimal subset of *db.DB used by readProfileConfigs.
// It keeps the function unit-testable.
type profileFetcher interface {
	GetProfileByName(name string) (db.Profile, error)
}

// readProfileConfigs loads profile rows from the DB and parses the elbencho
// INI ConfigJSON to extract block size, R/W ratio, pattern, runtime. It also
// parses ThresholdsJSON for the validation thresholds. Profiles whose row
// is missing or unparsable are returned with the name only and dashes for
// the other fields (handled in the template).
//
// The first parameter (profilesDir) is kept for plan compliance and future
// filesystem-based fallback, but is currently unused: profiles live in DB.
func readProfileConfigs(profilesDir string, fetcher profileFetcher, names []string) []ProfileConfig {
	_ = profilesDir
	out := make([]ProfileConfig, 0, len(names))
	for _, name := range names {
		pc := ProfileConfig{Name: name}
		if fetcher == nil {
			out = append(out, pc)
			continue
		}
		prof, err := fetcher.GetProfileByName(name)
		if err != nil {
			out = append(out, pc)
			continue
		}
		applyElbenchoConfig(&pc, prof.ConfigJSON)
		applyThresholds(&pc, prof.ThresholdsJSON)
		out = append(out, pc)
	}
	return out
}

// applyElbenchoConfig parses the elbencho INI text and fills BlockSize,
// RWRatio, Pattern, RuntimeSec on pc when found.
func applyElbenchoConfig(pc *ProfileConfig, configText string) {
	if strings.TrimSpace(configText) == "" {
		return
	}
	kv := parseINIKV(configText)
	if v, ok := kv["block"]; ok {
		pc.BlockSize = v
	}
	if v, ok := kv["timelimit"]; ok {
		if n, err := strconv.Atoi(v); err == nil {
			pc.RuntimeSec = n
		}
	}
	// elbencho ratio: "rwmixpct" is the read percent, or rand=1 + write phase.
	// We do best-effort: rwmixpct -> ratio; fall back to "100% R" / "100% W"
	// based on phase flags.
	if v, ok := kv["rwmixpct"]; ok {
		if n, err := strconv.Atoi(v); err == nil {
			pc.RWRatio = ratioLabel(n)
		}
	} else {
		// Look for write/read phase flags
		_, hasW := kv["w"]
		_, hasR := kv["r"]
		switch {
		case hasW && !hasR:
			pc.RWRatio = "100% W"
		case hasR && !hasW:
			pc.RWRatio = "100% R"
		}
	}
	if v, ok := kv["rand"]; ok && v == "1" {
		pc.Pattern = "random"
	} else if pc.Pattern == "" {
		// elbencho default is sequential when --rand is not set
		pc.Pattern = "sequential"
	}
}

func applyThresholds(pc *ProfileConfig, thresholdsJSON string) {
	if strings.TrimSpace(thresholdsJSON) == "" {
		return
	}
	var t db.ProfileThresholds
	if err := json.Unmarshal([]byte(thresholdsJSON), &t); err != nil {
		return
	}
	if t.MinIOPSRead > 0 {
		pc.ThresholdIOPSR = int(t.MinIOPSRead)
	}
	if t.MinIOPSWrite > 0 {
		pc.ThresholdIOPSW = int(t.MinIOPSWrite)
	}
	if t.MaxLatencyMs > 0 {
		pc.ThresholdLatencyMs = t.MaxLatencyMs
	}
}

// parseINIKV returns a key->value map from a flat INI / shell style text.
// Lines starting with '#' or empty lines are ignored. Section headers
// (e.g. "[base]") are ignored. "key=value" pairs are captured.
func parseINIKV(s string) map[string]string {
	out := map[string]string{}
	for _, raw := range strings.Split(s, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			continue
		}
		idx := strings.Index(line, "=")
		if idx <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		out[key] = val
	}
	return out
}

func ratioLabel(readPct int) string {
	switch {
	case readPct >= 100:
		return "100% R"
	case readPct <= 0:
		return "100% W"
	default:
		return strconv.Itoa(readPct) + "/" + strconv.Itoa(100-readPct) + " R/W"
	}
}
