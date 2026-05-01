-- Migration 013: persist worker hardware fields on jobs for methodology rendering.
-- Additive only. Existing rows will have NULL; use COALESCE in SELECT queries.

ALTER TABLE jobs ADD COLUMN worker_cpu       INTEGER;
ALTER TABLE jobs ADD COLUMN worker_ram_mb    INTEGER;
ALTER TABLE jobs ADD COLUMN data_disks       INTEGER;
ALTER TABLE jobs ADD COLUMN data_disk_gb     INTEGER;
ALTER TABLE jobs ADD COLUMN storage_pool     TEXT;
ALTER TABLE jobs ADD COLUMN proxmox_nodes_csv TEXT;
