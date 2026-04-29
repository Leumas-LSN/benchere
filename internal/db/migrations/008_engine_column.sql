-- v1.11.0: dual-engine support (fio alongside elbencho)
--
-- Adds an engine column to benchmark_profiles, jobs and results so the
-- backend can route a job to the right runner and the report can label
-- which engine produced the numbers. Existing rows default to engine
-- elbencho to keep historical reports coherent.
--
-- For benchmark_profiles we also need same name across engines (the six
-- canonical profile names live in both fio and elbencho), so we drop the
-- UNIQUE constraint on name and replace it with a composite unique on
-- (name, engine). SQLite cannot ALTER a UNIQUE constraint in place, so
-- we recreate the table.

-- 1. Recreate benchmark_profiles with composite uniqueness on (name, engine).
CREATE TABLE benchmark_profiles_new (
    id              TEXT PRIMARY KEY,
    name            TEXT NOT NULL,
    engine          TEXT NOT NULL DEFAULT 'elbencho',
    config_json     TEXT NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    thresholds_json TEXT NOT NULL DEFAULT '',
    is_builtin      INTEGER NOT NULL DEFAULT 0,
    UNIQUE(name, engine)
);

INSERT INTO benchmark_profiles_new (id, name, engine, config_json, description, thresholds_json, is_builtin)
SELECT id, name, 'elbencho', config_json, description, thresholds_json, is_builtin
FROM benchmark_profiles;

DROP TABLE benchmark_profiles;
ALTER TABLE benchmark_profiles_new RENAME TO benchmark_profiles;

-- 2. Add engine column to jobs and results so we know which engine produced
--    each row. Existing rows default to elbencho.
ALTER TABLE jobs    ADD COLUMN engine TEXT NOT NULL DEFAULT 'elbencho';
ALTER TABLE results ADD COLUMN engine TEXT NOT NULL DEFAULT 'elbencho';

-- 3. Seed six fio profiles equivalent to the elbencho ones. Same names so
--    the user picks the same workload regardless of engine. Same thresholds
--    where defined (copied from the elbencho row of the same name).
INSERT INTO benchmark_profiles (id, name, engine, config_json, description, thresholds_json, is_builtin)
SELECT
    lower(hex(randomblob(16))),
    'seq-256k-read',
    'fio',
    '[global]
ioengine=libaio
direct=1
bs=256k
iodepth=128
numjobs=1
runtime=300
time_based=1
group_reporting=1
filename=/dev/disk/by-id/scsi-0QEMU_QEMU_HARDDISK_drive-scsi1
refill_buffers=1
buffer_compress_percentage=0

[seq-256k-read]
rw=read
',
    'Read seq 256k - Restore, analytics, replication, debit lecture max.',
    COALESCE((SELECT thresholds_json FROM benchmark_profiles WHERE name = 'seq-256k-read' AND engine = 'elbencho'), ''),
    1;

INSERT INTO benchmark_profiles (id, name, engine, config_json, description, thresholds_json, is_builtin)
SELECT
    lower(hex(randomblob(16))),
    'seq-256k-write',
    'fio',
    '[global]
ioengine=libaio
direct=1
bs=256k
iodepth=128
numjobs=1
runtime=300
time_based=1
group_reporting=1
filename=/dev/disk/by-id/scsi-0QEMU_QEMU_HARDDISK_drive-scsi1
refill_buffers=1
buffer_compress_percentage=0

[seq-256k-write]
rw=write
',
    'Write seq 256k - Backup, snapshots, streaming video, debit ecriture max.',
    COALESCE((SELECT thresholds_json FROM benchmark_profiles WHERE name = 'seq-256k-write' AND engine = 'elbencho'), ''),
    1;

INSERT INTO benchmark_profiles (id, name, engine, config_json, description, thresholds_json, is_builtin)
SELECT
    lower(hex(randomblob(16))),
    'rand-4k-read',
    'fio',
    '[global]
ioengine=libaio
direct=1
bs=4k
iodepth=128
numjobs=1
runtime=180
time_based=1
group_reporting=1
filename=/dev/disk/by-id/scsi-0QEMU_QEMU_HARDDISK_drive-scsi1
refill_buffers=1
buffer_compress_percentage=0

[rand-4k-read]
rw=randread
',
    'Read random 4k - VDI cold-cache, boot storm, pire cas read.',
    COALESCE((SELECT thresholds_json FROM benchmark_profiles WHERE name = 'rand-4k-read' AND engine = 'elbencho'), ''),
    1;

INSERT INTO benchmark_profiles (id, name, engine, config_json, description, thresholds_json, is_builtin)
SELECT
    lower(hex(randomblob(16))),
    'rand-4k-write',
    'fio',
    '[global]
ioengine=libaio
direct=1
bs=4k
iodepth=128
numjobs=1
runtime=180
time_based=1
group_reporting=1
filename=/dev/disk/by-id/scsi-0QEMU_QEMU_HARDDISK_drive-scsi1
refill_buffers=1
buffer_compress_percentage=0

[rand-4k-write]
rw=randwrite
',
    'Write random 4k - OLTP write-heavy (INSERT/UPDATE intensifs en base transactionnelle).',
    COALESCE((SELECT thresholds_json FROM benchmark_profiles WHERE name = 'rand-4k-write' AND engine = 'elbencho'), ''),
    1;

INSERT INTO benchmark_profiles (id, name, engine, config_json, description, thresholds_json, is_builtin)
SELECT
    lower(hex(randomblob(16))),
    'rand-4k-70r30w',
    'fio',
    '[global]
ioengine=libaio
direct=1
bs=4k
iodepth=128
numjobs=1
runtime=180
time_based=1
group_reporting=1
filename=/dev/disk/by-id/scsi-0QEMU_QEMU_HARDDISK_drive-scsi1
refill_buffers=1
buffer_compress_percentage=0

[rand-4k-70r30w]
rw=randrw
rwmixread=70
',
    'Mixte 70/30 random 4k - OLTP typique en production (e-commerce, app metier).',
    COALESCE((SELECT thresholds_json FROM benchmark_profiles WHERE name = 'rand-4k-70r30w' AND engine = 'elbencho'), ''),
    1;

INSERT INTO benchmark_profiles (id, name, engine, config_json, description, thresholds_json, is_builtin)
SELECT
    lower(hex(randomblob(16))),
    'rand-8k-50r50w',
    'fio',
    '[global]
ioengine=libaio
direct=1
bs=8k
iodepth=128
numjobs=1
runtime=180
time_based=1
group_reporting=1
filename=/dev/disk/by-id/scsi-0QEMU_QEMU_HARDDISK_drive-scsi1
refill_buffers=1
buffer_compress_percentage=0

[rand-8k-50r50w]
rw=randrw
rwmixread=50
',
    'Mixte 50/50 random 8k - Bases de donnees mixtes (PostgreSQL, MySQL, SQL Server).',
    COALESCE((SELECT thresholds_json FROM benchmark_profiles WHERE name = 'rand-8k-50r50w' AND engine = 'elbencho'), ''),
    1;
