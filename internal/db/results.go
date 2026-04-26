package db

import "time"

func (d *DB) InsertResult(r Result) error {
	_, err := d.Exec(
		`INSERT INTO results(id,job_id,profile_name,timestamp,iops_read,iops_write,
		throughput_read_mbps,throughput_write_mbps,latency_avg_ms,latency_p99_ms)
		VALUES(?,?,?,?,?,?,?,?,?,?)`,
		r.ID, r.JobID, r.ProfileName, r.Timestamp,
		r.IOPSRead, r.IOPSWrite, r.ThroughputReadMBps, r.ThroughputWriteMBps,
		r.LatencyAvgMs, r.LatencyP99Ms,
	)
	return err
}

func (d *DB) InsertProxmoxSnapshot(s ProxmoxSnapshot) error {
	_, err := d.Exec(
		"INSERT INTO proxmox_snapshots(id,job_id,timestamp,node_name,cpu_pct,ram_pct,load_avg) VALUES(?,?,?,?,?,?,?)",
		s.ID, s.JobID, s.Timestamp, s.NodeName, s.CPUPct, s.RAMPct, s.LoadAvg,
	)
	return err
}

func (d *DB) InsertProxmoxVMSnapshot(s ProxmoxVMSnapshot) error {
	_, err := d.Exec(
		"INSERT INTO proxmox_vm_snapshots(id,job_id,timestamp,worker_id,cpu_pct) VALUES(?,?,?,?,?)",
		s.ID, s.JobID, s.Timestamp, s.WorkerID, s.CPUPct,
	)
	return err
}

func (d *DB) ListResultsByJob(jobID string) ([]Result, error) {
	rows, err := d.Query(
		`SELECT id,job_id,profile_name,timestamp,iops_read,iops_write,
		throughput_read_mbps,throughput_write_mbps,latency_avg_ms,latency_p99_ms
		FROM results WHERE job_id=? ORDER BY timestamp`,
		jobID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []Result
	for rows.Next() {
		var r Result
		if err := rows.Scan(&r.ID, &r.JobID, &r.ProfileName, &r.Timestamp,
			&r.IOPSRead, &r.IOPSWrite, &r.ThroughputReadMBps, &r.ThroughputWriteMBps,
			&r.LatencyAvgMs, &r.LatencyP99Ms); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

func (d *DB) ListProxmoxSnapshotsByJob(jobID string) ([]ProxmoxSnapshot, error) {
	rows, err := d.Query(
		"SELECT id,job_id,timestamp,node_name,cpu_pct,ram_pct,load_avg FROM proxmox_snapshots WHERE job_id=? ORDER BY timestamp",
		jobID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var snaps []ProxmoxSnapshot
	for rows.Next() {
		var s ProxmoxSnapshot
		if err := rows.Scan(&s.ID, &s.JobID, &s.Timestamp, &s.NodeName, &s.CPUPct, &s.RAMPct, &s.LoadAvg); err != nil {
			return nil, err
		}
		snaps = append(snaps, s)
	}
	return snaps, rows.Err()
}

func (d *DB) ListProxmoxSnapshotsSince(jobID string, since time.Time) ([]ProxmoxSnapshot, error) {
	rows, err := d.Query(
		"SELECT id,job_id,timestamp,node_name,cpu_pct,ram_pct,load_avg FROM proxmox_snapshots WHERE job_id=? AND timestamp>=? ORDER BY timestamp",
		jobID, since,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var snaps []ProxmoxSnapshot
	for rows.Next() {
		var s ProxmoxSnapshot
		if err := rows.Scan(&s.ID, &s.JobID, &s.Timestamp, &s.NodeName, &s.CPUPct, &s.RAMPct, &s.LoadAvg); err != nil {
			return nil, err
		}
		snaps = append(snaps, s)
	}
	return snaps, rows.Err()
}
