-- Rename the 6 default profiles to a clearer technical convention and tune
-- iodepth + timelimit. Old jobs keep referencing the old names because the
-- results.profile_name column is TEXT (no FK) so historical reports remain
-- readable.
--
-- Naming scheme: <pattern>-<blocksize>-<role>
--   pattern   : seq | rand
--   blocksize : 4k | 8k | 256k
--   role      : write | read | <pct>r<pct>w (mixed)
--
-- Tuning:
--   iodepth=4 -> iodepth=16 on every profile. With 8 threads x 16 depth on
--     9 workers = 1152 in-flight IO, enough to saturate Ceph NVMe without
--     overwhelming the 25 Gbps mesh.
--   timelimit=300 -> timelimit=180 on the 4 random profiles. They reach
--     steady state quickly. Sequential profiles keep 300 s to fully fill
--     the read-ahead and writeback pipelines.

UPDATE benchmark_profiles
SET name = 'rand-4k-write',
    description = 'Write random 4k - OLTP write-heavy (INSERT/UPDATE intensifs en base transactionnelle).',
    config_json = REPLACE(REPLACE(config_json, 'iodepth=4', 'iodepth=16'), 'timelimit=300', 'timelimit=180')
WHERE name = '4k_0read_100random';

UPDATE benchmark_profiles
SET name = 'rand-4k-70r30w',
    description = 'Mixte 70/30 random 4k - OLTP typique en production (e-commerce, app metier).',
    config_json = REPLACE(REPLACE(config_json, 'iodepth=4', 'iodepth=16'), 'timelimit=300', 'timelimit=180')
WHERE name = '4k_70read_100random';

UPDATE benchmark_profiles
SET name = 'rand-4k-read',
    description = 'Read random 4k - VDI cold-cache, boot storm, pire cas read.',
    config_json = REPLACE(REPLACE(config_json, 'iodepth=4', 'iodepth=16'), 'timelimit=300', 'timelimit=180')
WHERE name = '4k_100read_100random';

UPDATE benchmark_profiles
SET name = 'rand-8k-50r50w',
    description = 'Mixte 50/50 random 8k - Bases de donnees mixtes (PostgreSQL, MySQL, SQL Server).',
    config_json = REPLACE(REPLACE(config_json, 'iodepth=4', 'iodepth=16'), 'timelimit=300', 'timelimit=180')
WHERE name = '8k_50read_100random';

UPDATE benchmark_profiles
SET name = 'seq-256k-write',
    description = 'Write seq 256k - Backup, snapshots, streaming video, debit ecriture max.',
    config_json = REPLACE(config_json, 'iodepth=4', 'iodepth=16')
WHERE name = '256k_0read_0random';

UPDATE benchmark_profiles
SET name = 'seq-256k-read',
    description = 'Read seq 256k - Restore, analytics, replication, debit lecture max.',
    config_json = REPLACE(config_json, 'iodepth=4', 'iodepth=16')
WHERE name = '256k_100read_0random';
