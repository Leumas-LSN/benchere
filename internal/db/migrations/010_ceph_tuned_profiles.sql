-- v1.12.2: ceph-tuned fio profiles (numjobs=8 iodepth=32 offset_increment=2G).
--
-- The default fio profiles use numjobs=1 iodepth=128 - good for local NVMe
-- where a single submission thread can keep the queue full, but too thin
-- for high-latency clustered storage like Ceph RBD where each IO costs
-- 1-2ms (network + replica + journal). One thread caps at iodepth/latency
-- effective IOPS, leaving the cluster idle.
--
-- These 5 new profiles parallelise the work across 8 fio processes per
-- worker, each with iodepth=32 (still 256 IOs in flight per worker, like
-- before), and stagger their offsets by 2G so they hit different RADOS
-- placement groups instead of the same hot blocks at offset 0. This is
-- the HCIBench / Ceph upstream pattern for cluster-level saturation.
--
-- Profiles are added alongside the originals (suffixed -ceph), so the
-- non-ceph defaults remain available for local-disk scenarios.

INSERT INTO benchmark_profiles (id, name, engine, config_json, description, thresholds_json, is_builtin)
VALUES (
    lower(hex(randomblob(16))),
    'rand-4k-read-ceph',
    'fio',
    '[global]
ioengine=libaio
direct=1
bs=4k
iodepth=32
numjobs=8
offset_increment=2G
runtime=180
time_based=1
group_reporting=1
filename=/dev/disk/by-id/scsi-0QEMU_QEMU_HARDDISK_drive-scsi1
refill_buffers=1
buffer_compress_percentage=0

[rand-4k-read-ceph]
rw=randread',
    'Random 4K read tuned for Ceph RBD: 8 jobs x iodepth 32 with 2G offset stride.',
    '',
    1
);

INSERT INTO benchmark_profiles (id, name, engine, config_json, description, thresholds_json, is_builtin)
VALUES (
    lower(hex(randomblob(16))),
    'rand-4k-write-ceph',
    'fio',
    '[global]
ioengine=libaio
direct=1
bs=4k
iodepth=32
numjobs=8
offset_increment=2G
runtime=180
time_based=1
group_reporting=1
filename=/dev/disk/by-id/scsi-0QEMU_QEMU_HARDDISK_drive-scsi1
refill_buffers=1
buffer_compress_percentage=0

[rand-4k-write-ceph]
rw=randwrite',
    'Random 4K write tuned for Ceph RBD: 8 jobs x iodepth 32 with 2G offset stride.',
    '',
    1
);

INSERT INTO benchmark_profiles (id, name, engine, config_json, description, thresholds_json, is_builtin)
VALUES (
    lower(hex(randomblob(16))),
    'rand-4k-70r30w-ceph',
    'fio',
    '[global]
ioengine=libaio
direct=1
bs=4k
iodepth=32
numjobs=8
offset_increment=2G
runtime=180
time_based=1
group_reporting=1
filename=/dev/disk/by-id/scsi-0QEMU_QEMU_HARDDISK_drive-scsi1
refill_buffers=1
buffer_compress_percentage=0

[rand-4k-70r30w-ceph]
rw=randrw
rwmixread=70',
    'Random 4K mixed 70/30 tuned for Ceph RBD: 8 jobs x iodepth 32 with 2G offset stride.',
    '',
    1
);

INSERT INTO benchmark_profiles (id, name, engine, config_json, description, thresholds_json, is_builtin)
VALUES (
    lower(hex(randomblob(16))),
    'seq-256k-read-ceph',
    'fio',
    '[global]
ioengine=libaio
direct=1
bs=256k
iodepth=64
numjobs=4
offset_increment=4G
runtime=300
time_based=1
group_reporting=1
filename=/dev/disk/by-id/scsi-0QEMU_QEMU_HARDDISK_drive-scsi1
refill_buffers=1
buffer_compress_percentage=0

[seq-256k-read-ceph]
rw=read',
    'Sequential 256K read tuned for Ceph RBD: 4 jobs x iodepth 64 with 4G offset stride.',
    '',
    1
);

INSERT INTO benchmark_profiles (id, name, engine, config_json, description, thresholds_json, is_builtin)
VALUES (
    lower(hex(randomblob(16))),
    'seq-256k-write-ceph',
    'fio',
    '[global]
ioengine=libaio
direct=1
bs=256k
iodepth=64
numjobs=4
offset_increment=4G
runtime=300
time_based=1
group_reporting=1
filename=/dev/disk/by-id/scsi-0QEMU_QEMU_HARDDISK_drive-scsi1
refill_buffers=1
buffer_compress_percentage=0

[seq-256k-write-ceph]
rw=write',
    'Sequential 256K write tuned for Ceph RBD: 4 jobs x iodepth 64 with 4G offset stride.',
    '',
    1
);
