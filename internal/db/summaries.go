package db

import "time"

func (d *DB) InsertPhaseSummary(s PhaseSummary) error {
	_, err := d.Exec(
		`INSERT INTO phase_summaries(
			id, job_id, profile_name, samples_count,
			iops_read_avg, iops_read_min, iops_read_max,
			iops_write_avg, iops_write_min, iops_write_max,
			throughput_read_mbps_avg, throughput_read_mbps_max,
			throughput_write_mbps_avg, throughput_write_mbps_max,
			lat_p50_ms, lat_p95_ms, lat_p99_ms, lat_p999_ms,
			lat_write_p99_ms, iops_cv_pct, finished_at
		) VALUES(?,?,?,?, ?,?,?, ?,?,?, ?,?, ?,?, ?,?,?,?, ?,?,?)
		ON CONFLICT(job_id, profile_name) DO UPDATE SET
			samples_count = excluded.samples_count,
			iops_read_avg = excluded.iops_read_avg,
			iops_read_min = excluded.iops_read_min,
			iops_read_max = excluded.iops_read_max,
			iops_write_avg = excluded.iops_write_avg,
			iops_write_min = excluded.iops_write_min,
			iops_write_max = excluded.iops_write_max,
			throughput_read_mbps_avg = excluded.throughput_read_mbps_avg,
			throughput_read_mbps_max = excluded.throughput_read_mbps_max,
			throughput_write_mbps_avg = excluded.throughput_write_mbps_avg,
			throughput_write_mbps_max = excluded.throughput_write_mbps_max,
			lat_p50_ms = excluded.lat_p50_ms,
			lat_p95_ms = excluded.lat_p95_ms,
			lat_p99_ms = excluded.lat_p99_ms,
			lat_p999_ms = excluded.lat_p999_ms,
			lat_write_p99_ms = excluded.lat_write_p99_ms,
			iops_cv_pct = excluded.iops_cv_pct,
			finished_at = excluded.finished_at`,
		s.ID, s.JobID, s.ProfileName, s.SamplesCount,
		s.IOPSReadAvg, s.IOPSReadMin, s.IOPSReadMax,
		s.IOPSWriteAvg, s.IOPSWriteMin, s.IOPSWriteMax,
		s.ThroughputReadMBpsAvg, s.ThroughputReadMBpsMax,
		s.ThroughputWriteMBpsAvg, s.ThroughputWriteMBpsMax,
		s.LatP50Ms, s.LatP95Ms, s.LatP99Ms, s.LatP999Ms,
		s.LatWriteP99Ms, s.IOPSCVPct, s.FinishedAt,
	)
	return err
}

func (d *DB) ListPhaseSummariesByJob(jobID string) ([]PhaseSummary, error) {
	rows, err := d.Query(
		`SELECT id, job_id, profile_name, samples_count,
			iops_read_avg, iops_read_min, iops_read_max,
			iops_write_avg, iops_write_min, iops_write_max,
			throughput_read_mbps_avg, throughput_read_mbps_max,
			throughput_write_mbps_avg, throughput_write_mbps_max,
			lat_p50_ms, lat_p95_ms, lat_p99_ms, lat_p999_ms,
			lat_write_p99_ms, iops_cv_pct, finished_at
		FROM phase_summaries WHERE job_id=? ORDER BY finished_at`,
		jobID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PhaseSummary
	for rows.Next() {
		var s PhaseSummary
		var ts time.Time
		if err := rows.Scan(
			&s.ID, &s.JobID, &s.ProfileName, &s.SamplesCount,
			&s.IOPSReadAvg, &s.IOPSReadMin, &s.IOPSReadMax,
			&s.IOPSWriteAvg, &s.IOPSWriteMin, &s.IOPSWriteMax,
			&s.ThroughputReadMBpsAvg, &s.ThroughputReadMBpsMax,
			&s.ThroughputWriteMBpsAvg, &s.ThroughputWriteMBpsMax,
			&s.LatP50Ms, &s.LatP95Ms, &s.LatP99Ms, &s.LatP999Ms,
			&s.LatWriteP99Ms, &s.IOPSCVPct, &ts,
		); err != nil {
			return nil, err
		}
		s.FinishedAt = ts
		out = append(out, s)
	}
	return out, rows.Err()
}
