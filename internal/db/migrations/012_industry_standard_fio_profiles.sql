-- Migration 012: industry-standard fio profiles for v2.0.0.
-- These supplement the existing ceph-tuned profiles seeded by 010 (which
-- are renamed with a ceph- prefix in this migration so the UI can group
-- them as "Ceph-tuned" while these stay under "Standard").

-- Step 1: rename existing ceph-tuned fio profiles for clarity in the picker.
UPDATE benchmark_profiles
SET name = 'ceph-' || name
WHERE engine = 'fio'
  AND is_builtin = 1
  AND name NOT LIKE 'ceph-%'
  AND name NOT LIKE 'oltp-%'
  AND name NOT LIKE 'sql-%'
  AND name NOT LIKE 'vdi-%'
  AND name NOT LIKE 'backup-%'
  AND name NOT LIKE 'mixed-%';

-- Step 2: insert the six industry-standard profiles. Each carries:
-- - runtime=300 (5 min steady-state)
-- - ramp_time=30 (30 s warmup, excluded from metrics)
-- - direct=1, ioengine=libaio
-- - clat_percentiles=1, percentile_list=50:95:99:99.9
-- - thresholds aligned for a baseline NVMe SSD setup; user can edit.

INSERT OR IGNORE INTO benchmark_profiles (id, name, engine, config_json, description, thresholds_json, is_builtin)
VALUES
  ('std-fio-oltp-4k-70-30',
   'oltp-4k-70-30',
   'fio',
   '[global]
ioengine=libaio
direct=1
filename=<TARGET>
bs=4k
rw=randrw
rwmixread=70
iodepth=4
numjobs=4
group_reporting=1
runtime=300
ramp_time=30
time_based=1
clat_percentiles=1
percentile_list=50:95:99:99.9
norandommap=1

[oltp-4k-70-30]
',
   'OLTP / DB workload. 4K random, 70R/30W, QD=4. Industry-standard reference equivalent to HCIBench Easy Run base profile. The most cited profile for transactional storage validation.',
   '{"min_iops_read":50000,"min_iops_write":20000,"max_p99_latency_ms":5.0,"max_avg_latency_ms":1.0}',
   1),

  ('std-fio-sql-8k-70-30',
   'sql-8k-70-30',
   'fio',
   '[global]
ioengine=libaio
direct=1
filename=<TARGET>
bs=8k
rw=randrw
rwmixread=70
iodepth=8
numjobs=4
group_reporting=1
runtime=300
ramp_time=30
time_based=1
clat_percentiles=1
percentile_list=50:95:99:99.9
norandommap=1

[sql-8k-70-30]
',
   'SQL Server typical workload. 8K random, 70R/30W, QD=8. Defensible reference for relational database storage validation.',
   '{"min_iops_read":30000,"min_iops_write":12000,"max_p99_latency_ms":8.0,"max_avg_latency_ms":2.0}',
   1),

  ('std-fio-vdi-4k-20-80',
   'vdi-4k-20-80',
   'fio',
   '[global]
ioengine=libaio
direct=1
filename=<TARGET>
bs=4k
rw=randrw
rwmixread=20
iodepth=8
numjobs=4
group_reporting=1
runtime=300
ramp_time=30
time_based=1
clat_percentiles=1
percentile_list=50:95:99:99.9
norandommap=1

[vdi-4k-20-80]
',
   'VDI burst pattern. 4K random, 20R/80W, QD=8. Write-heavy bursty workload matching VDI golden image boots and persistent desktop writes.',
   '{"min_iops_read":15000,"min_iops_write":50000,"max_p99_latency_ms":6.0,"max_avg_latency_ms":1.5}',
   1),

  ('std-fio-backup-256k-read',
   'backup-256k-read',
   'fio',
   '[global]
ioengine=libaio
direct=1
filename=<TARGET>
bs=256k
rw=read
iodepth=16
numjobs=4
group_reporting=1
runtime=180
ramp_time=15
time_based=1
clat_percentiles=1
percentile_list=50:95:99:99.9

[backup-256k-read]
',
   'Backup throughput. 256K sequential read, QD=16. Reads at backup-window speed for capacity planning.',
   '{"min_iops_read":2000,"max_p99_latency_ms":50.0,"max_avg_latency_ms":20.0}',
   1),

  ('std-fio-backup-256k-write',
   'backup-256k-write',
   'fio',
   '[global]
ioengine=libaio
direct=1
filename=<TARGET>
bs=256k
rw=write
iodepth=16
numjobs=4
group_reporting=1
runtime=180
ramp_time=15
time_based=1
clat_percentiles=1
percentile_list=50:95:99:99.9

[backup-256k-write]
',
   'Restore throughput. 256K sequential write, QD=16. Restore from backup at full bandwidth.',
   '{"min_iops_write":2000,"max_p99_latency_ms":50.0,"max_avg_latency_ms":20.0}',
   1),

  ('std-fio-mixed-32k-50-50',
   'mixed-32k-50-50',
   'fio',
   '[global]
ioengine=libaio
direct=1
filename=<TARGET>
bs=32k
rw=randrw
rwmixread=50
iodepth=8
numjobs=4
group_reporting=1
runtime=300
ramp_time=30
time_based=1
clat_percentiles=1
percentile_list=50:95:99:99.9
norandommap=1

[mixed-32k-50-50]
',
   'Generic mixed workload. 32K random, 50R/50W, QD=8. Average application I/O pattern.',
   '{"min_iops_read":15000,"min_iops_write":15000,"max_p99_latency_ms":10.0,"max_avg_latency_ms":3.0}',
   1);
