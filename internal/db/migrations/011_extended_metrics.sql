-- Migration 011: extended fio percentile metrics + phase summaries.
-- Additive only. Existing rows are preserved; new columns nullable.

ALTER TABLE results ADD COLUMN latency_read_avg_ms  REAL;
ALTER TABLE results ADD COLUMN latency_write_avg_ms REAL;
ALTER TABLE results ADD COLUMN latency_p50_ms       REAL;
ALTER TABLE results ADD COLUMN latency_p95_ms       REAL;
ALTER TABLE results ADD COLUMN latency_p999_ms      REAL;
ALTER TABLE results ADD COLUMN latency_write_p99_ms REAL;

CREATE TABLE IF NOT EXISTS phase_summaries (
    id                       TEXT PRIMARY KEY,
    job_id                   TEXT NOT NULL REFERENCES jobs(id),
    profile_name             TEXT NOT NULL,
    samples_count            INTEGER NOT NULL,
    iops_read_avg            REAL,
    iops_read_min            REAL,
    iops_read_max            REAL,
    iops_write_avg           REAL,
    iops_write_min           REAL,
    iops_write_max           REAL,
    throughput_read_mbps_avg REAL,
    throughput_read_mbps_max REAL,
    throughput_write_mbps_avg REAL,
    throughput_write_mbps_max REAL,
    lat_p50_ms               REAL,
    lat_p95_ms               REAL,
    lat_p99_ms               REAL,
    lat_p999_ms              REAL,
    lat_write_p99_ms         REAL,
    iops_cv_pct              REAL,
    finished_at              DATETIME NOT NULL,
    UNIQUE(job_id, profile_name)
);

CREATE INDEX IF NOT EXISTS idx_phase_summaries_job ON phase_summaries(job_id);
