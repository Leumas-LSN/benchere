CREATE TABLE IF NOT EXISTS settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS benchmark_profiles (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL UNIQUE,
    config_json TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS jobs (
    id           TEXT PRIMARY KEY,
    name         TEXT NOT NULL,
    client_name  TEXT NOT NULL,
    status       TEXT NOT NULL DEFAULT 'pending',
    mode         TEXT NOT NULL,
    created_at   DATETIME NOT NULL,
    finished_at  DATETIME
);

CREATE TABLE IF NOT EXISTS workers (
    id            TEXT PRIMARY KEY,
    job_id        TEXT NOT NULL REFERENCES jobs(id),
    vm_id         INTEGER NOT NULL,
    proxmox_node  TEXT NOT NULL,
    ip            TEXT,
    status        TEXT NOT NULL DEFAULT 'provisioning'
);

CREATE TABLE IF NOT EXISTS results (
    id                    TEXT PRIMARY KEY,
    job_id                TEXT NOT NULL REFERENCES jobs(id),
    profile_name          TEXT NOT NULL,
    timestamp             DATETIME NOT NULL,
    iops_read             REAL,
    iops_write            REAL,
    throughput_read_mbps  REAL,
    throughput_write_mbps REAL,
    latency_avg_ms        REAL,
    latency_p99_ms        REAL
);

CREATE TABLE IF NOT EXISTS proxmox_snapshots (
    id         TEXT PRIMARY KEY,
    job_id     TEXT NOT NULL REFERENCES jobs(id),
    timestamp  DATETIME NOT NULL,
    node_name  TEXT NOT NULL,
    cpu_pct    REAL,
    ram_pct    REAL,
    load_avg   REAL
);

CREATE TABLE IF NOT EXISTS proxmox_vm_snapshots (
    id         TEXT PRIMARY KEY,
    job_id     TEXT NOT NULL REFERENCES jobs(id),
    timestamp  DATETIME NOT NULL,
    worker_id  TEXT NOT NULL REFERENCES workers(id),
    cpu_pct    REAL
);
