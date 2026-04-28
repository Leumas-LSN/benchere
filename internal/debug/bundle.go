package debug

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Leumas-LSN/benchere/internal/db"
	"github.com/Leumas-LSN/benchere/internal/proxmox"
)

// BundleSources holds the inputs the assembler needs to gather artifacts.
type BundleSources struct {
	DB             *db.DB
	Proxmox        *proxmox.Client
	JobsDir        string // e.g. /var/lib/benchere/jobs
	Version        string
	JobID          string
	IncludeJournal bool // false in tests
	IncludeProxmox bool // false in tests / when no client
	IncludeCeph    bool
	IncludeDBCopy  bool
}

// Build streams a tar.gz of the bundle into w. The job must exist; the caller
// already validated job.Status. Errors that affect a single collector are
// recorded into errors.log inside the archive instead of aborting.
func Build(ctx context.Context, w io.Writer, src BundleSources) error {
	job, err := src.DB.GetJob(src.JobID)
	if err != nil {
		return fmt.Errorf("get job: %w", err)
	}

	gz := gzip.NewWriter(w)
	defer gz.Close()
	tw := tar.NewWriter(gz)
	defer tw.Close()

	now := time.Now()
	var errs []string
	addErr := func(msg string) { errs = append(errs, fmt.Sprintf("%s %s", now.UTC().Format(time.RFC3339), msg)) }

	// Track everything we add for the manifest.
	var entries []ManifestEntry
	writeFile := func(path string, data []byte) error {
		hdr := &tar.Header{
			Name:    path,
			Mode:    0o644,
			Size:    int64(len(data)),
			ModTime: now,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if _, err := tw.Write(data); err != nil {
			return err
		}
		entries = append(entries, ManifestEntry{Path: path, Size: int64(len(data))})
		return nil
	}

	// 1. README
	if err := writeFile("README.txt", []byte(BuildReadme(src.Version))); err != nil {
		return err
	}

	// 2. version.txt
	if err := writeFile("benchere/version.txt",
		[]byte(fmt.Sprintf("Benchere %s\n", src.Version))); err != nil {
		return err
	}

	// 3. settings (scrubbed)
	settings, err := src.DB.AllSettings()
	if err != nil {
		addErr("settings: " + err.Error())
		settings = map[string]string{}
	}
	cleaned, err := ScrubSettingsJSON(settings)
	if err != nil {
		addErr("settings json: " + err.Error())
		cleaned = []byte("{}")
	}
	if err := writeFile("benchere/settings.json", cleaned); err != nil {
		return err
	}

	// 4. DB snapshot via VACUUM INTO (consistent point-in-time copy).
	if src.IncludeDBCopy {
		dbBytes, dberr := snapshotDB(src.DB)
		if dberr != nil {
			addErr("db snapshot: " + dberr.Error())
			dbBytes = []byte("# unavailable: " + dberr.Error() + "\n")
		}
		if err := writeFile("benchere/db.sqlite", dbBytes); err != nil {
			return err
		}
	}

	// 5. journalctl
	if src.IncludeJournal {
		var fin time.Time
		if job.FinishedAt != nil {
			fin = *job.FinishedAt
		}
		jctl := CaptureJournal(ctx, job.CreatedAt, fin)
		if err := writeFile("benchere/journalctl.log", jctl); err != nil {
			return err
		}
	}

	// 6. job/ artifacts captured during the run from JobsDir.
	jobDir := filepath.Join(src.JobsDir, src.JobID)
	if err := walkDirInto(tw, jobDir, "job/", &entries, addErr, now); err != nil {
		addErr("walk job dir: " + err.Error())
	}

	// 7. results.csv from DB.
	resCSV, csverr := exportResultsCSV(src.DB, src.JobID)
	if csverr != nil {
		addErr("results csv: " + csverr.Error())
	} else {
		if err := writeFile("job/results.csv", resCSV); err != nil {
			return err
		}
	}

	// 8. metrics.csv (proxmox node + vm snapshots).
	mCSV := exportMetricsCSV(src.DB, src.JobID, addErr)
	if err := writeFile("job/metrics.csv", mCSV); err != nil {
		return err
	}

	// 9. Proxmox cluster snapshot
	if src.IncludeProxmox && src.Proxmox != nil {
		pc := &ProxmoxCollector{Client: src.Proxmox}
		if err := writeFile("proxmox/cluster-resources.json",
			pc.RawOrUnavailable(ctx, "/cluster/resources")); err != nil {
			return err
		}
		if err := writeFile("proxmox/storage-cfg.txt",
			[]byte(ScrubStorageCfgText(string(pc.StoragesJSON(ctx))))); err != nil {
			return err
		}
		if err := writeFile("proxmox/pveversion.txt", pc.PVEVersionJSON(ctx)); err != nil {
			return err
		}
		// Per-node statuses
		nodes, err := src.Proxmox.GetNodes(ctx)
		if err != nil {
			addErr("proxmox nodes: " + err.Error())
		}
		for _, n := range nodes {
			if err := writeFile(fmt.Sprintf("proxmox/nodes/%s/status.json", n),
				pc.NodeStatus(ctx, n)); err != nil {
				return err
			}
		}
	}

	// 10. Ceph best-effort
	if src.IncludeCeph && src.Proxmox != nil {
		cc := &CephCollector{Client: src.Proxmox}
		if err := writeFile("ceph/status.txt", cc.Status(ctx)); err != nil {
			return err
		}
		if err := writeFile("ceph/df.txt", cc.DF(ctx)); err != nil {
			return err
		}
		if err := writeFile("ceph/pools.txt", cc.Pools(ctx)); err != nil {
			return err
		}
		if err := writeFile("ceph/osd.txt", cc.OSD(ctx)); err != nil {
			return err
		}
		if err := writeFile("ceph/config-dump.txt", cc.Config(ctx)); err != nil {
			return err
		}
	}

	// 11. errors.log
	if len(errs) > 0 {
		body := strings.Join(errs, "\n") + "\n"
		if err := writeFile("errors.log", []byte(body)); err != nil {
			return err
		}
	}

	// 12. MANIFEST last (it counts itself out, header lines first then sorted entries)
	jobFin := time.Time{}
	if job.FinishedAt != nil {
		jobFin = *job.FinishedAt
	}
	manifest := BuildManifest(src.Version, src.JobID, job.CreatedAt, jobFin, now, entries)
	if err := writeFileLast(tw, "MANIFEST.txt", []byte(manifest), now); err != nil {
		return err
	}
	return nil
}

// writeFileLast writes one tar entry without growing the entries slice. Used
// for MANIFEST.txt which is built from the entries themselves.
func writeFileLast(tw *tar.Writer, path string, data []byte, when time.Time) error {
	hdr := &tar.Header{
		Name:    path,
		Mode:    0o644,
		Size:    int64(len(data)),
		ModTime: when,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	_, err := tw.Write(data)
	return err
}

// walkDirInto adds every regular file under root into the tar, prefixing the
// name with prefix. Hidden files starting with "." are skipped. Symlinks
// outside root are not followed. Missing root is silently ignored.
func walkDirInto(tw *tar.Writer, root, prefix string, entries *[]ManifestEntry, addErr func(string), now time.Time) error {
	st, err := os.Stat(root)
	if err != nil || !st.IsDir() {
		return nil
	}
	return filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			addErr(fmt.Sprintf("walk %s: %v", path, walkErr))
			return nil
		}
		if info.IsDir() {
			return nil
		}
		base := info.Name()
		if strings.HasPrefix(base, ".") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		archivePath := prefix + filepath.ToSlash(rel)
		data, err := os.ReadFile(path)
		if err != nil {
			addErr(fmt.Sprintf("read %s: %v", path, err))
			return nil
		}
		hdr := &tar.Header{
			Name:    archivePath,
			Mode:    0o644,
			Size:    int64(len(data)),
			ModTime: now,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if _, err := tw.Write(data); err != nil {
			return err
		}
		*entries = append(*entries, ManifestEntry{Path: archivePath, Size: int64(len(data))})
		return nil
	})
}

// snapshotDB returns the bytes of a fresh VACUUM INTO copy of the live DB.
// We use a temp file under os.TempDir() because VACUUM INTO needs a real
// path on disk; the file is removed before this returns.
func snapshotDB(d *db.DB) ([]byte, error) {
	tmp, err := os.CreateTemp("", "benchere-*.sqlite")
	if err != nil {
		return nil, err
	}
	tmpPath := tmp.Name()
	tmp.Close()
	_ = os.Remove(tmpPath) // VACUUM INTO requires the target not to exist.
	defer os.Remove(tmpPath)
	if _, err := d.Exec("VACUUM INTO ?", tmpPath); err != nil {
		return nil, err
	}
	return os.ReadFile(tmpPath)
}

// exportResultsCSV writes a header + rows for the given job, returning the
// CSV bytes. Returns a non-nil error only on DB failures; an empty result
// set is a valid empty CSV with just the header.
func exportResultsCSV(d *db.DB, jobID string) ([]byte, error) {
	rows, err := d.ListResultsByJob(jobID)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	cw := csv.NewWriter(&buf)
	_ = cw.Write([]string{
		"id", "job_id", "profile_name", "timestamp",
		"iops_read", "iops_write",
		"throughput_read_mbps", "throughput_write_mbps",
		"latency_avg_ms", "latency_p99_ms",
	})
	for _, r := range rows {
		_ = cw.Write([]string{
			r.ID, r.JobID, r.ProfileName,
			r.Timestamp.UTC().Format(time.RFC3339Nano),
			fmt.Sprintf("%.4f", r.IOPSRead),
			fmt.Sprintf("%.4f", r.IOPSWrite),
			fmt.Sprintf("%.4f", r.ThroughputReadMBps),
			fmt.Sprintf("%.4f", r.ThroughputWriteMBps),
			fmt.Sprintf("%.4f", r.LatencyAvgMs),
			fmt.Sprintf("%.4f", r.LatencyP99Ms),
		})
	}
	cw.Flush()
	if err := cw.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// exportMetricsCSV writes node and VM proxmox snapshots in a single CSV
// distinguished by a "kind" column. Errors are appended via addErr instead
// of aborting.
func exportMetricsCSV(d *db.DB, jobID string, addErr func(string)) []byte {
	var buf bytes.Buffer
	cw := csv.NewWriter(&buf)
	_ = cw.Write([]string{
		"kind", "id", "job_id", "timestamp", "subject",
		"cpu_pct", "ram_pct", "load_avg",
	})
	nodes, err := d.ListProxmoxSnapshotsByJob(jobID)
	if err != nil {
		addErr("metrics nodes: " + err.Error())
	}
	for _, s := range nodes {
		_ = cw.Write([]string{
			"node", s.ID, s.JobID,
			s.Timestamp.UTC().Format(time.RFC3339Nano),
			s.NodeName,
			fmt.Sprintf("%.4f", s.CPUPct),
			fmt.Sprintf("%.4f", s.RAMPct),
			fmt.Sprintf("%.4f", s.LoadAvg),
		})
	}
	cw.Flush()
	if err := cw.Error(); err != nil {
		addErr("metrics csv flush: " + err.Error())
	}
	return buf.Bytes()
}

// RawOrUnavailable returns the raw GET response or a stub describing the
// failure, used by the bundle for endpoints we do not model with a typed
// helper.
func (p *ProxmoxCollector) RawOrUnavailable(ctx context.Context, path string) []byte {
	if p == nil || p.Client == nil {
		return []byte("# unavailable: no proxmox client\n")
	}
	out, err := p.Client.RawGet(ctx, path)
	if err != nil {
		return []byte(fmt.Sprintf("# unavailable: %v\n", err))
	}
	return out
}
