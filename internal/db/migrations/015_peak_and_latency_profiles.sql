-- Migration 015: defensive ceph cleanup + add 6 backend-agnostic
-- characterization profiles (peak performance + latency floor).
--
-- Defensive cleanup: catch any *-ceph residuals that did not start with
-- ceph- and so escaped migration 014. No-op on a freshly-installed DB
-- but keeps the catalog clean if a future user runs a custom seed.
DELETE FROM benchmark_profiles WHERE engine = 'fio' AND name LIKE '%-ceph';
DELETE FROM benchmark_profiles WHERE engine = 'fio' AND name LIKE '%ceph%';

-- The 6 industry-standard profiles seeded by migration 012 stay
-- (oltp-4k-70-30, sql-8k-70-30, vdi-4k-20-80, backup-256k-read,
-- backup-256k-write, mixed-32k-50-50). They cover realistic workloads
-- and are appropriate for client SLA validation with thresholds.
--
-- The 6 profiles below extend the catalog with characterization
-- workloads that produce the upper bounds vendors publish: peak IOPS
-- (small block, max queue depth) and peak bandwidth (large block,
-- sequential), plus the responsiveness floor at QD=1.
--
-- All profiles are backend-agnostic: same fio jobfile must run on iSCSI,
-- NFS, DataCore, vSAN, Ceph RBD, LVM thin, NVMe local, or anything else
-- presenting a block device. No backend-specific tuning is allowed in
-- this catalog by product policy.
--
-- Thresholds are intentionally empty for characterization profiles
-- because the upper bound is by definition storage-dependent. Users can
-- add thresholds per-deployment via the API if they want pass/fail.

INSERT OR IGNORE INTO benchmark_profiles (id, name, engine, config_json, description, thresholds_json, is_builtin)
VALUES
  ('std-fio-peak-read-iops',
   'peak-read-iops',
   'fio',
   '[global]
ioengine=libaio
direct=1
filename=<TARGET>
bs=4k
rw=randread
iodepth=32
numjobs=4
group_reporting=1
runtime=180
ramp_time=30
time_based=1
clat_percentiles=1
percentile_list=50:95:99:99.9
norandommap=1

[peak-read-iops]
',
   'Peak read IOPS characterization. 4K random 100 percent read, QD=32, 4 jobs. Saturates the read path to expose the storage upper bound. Not a realistic workload, do not use for SLA acceptance.',
   '{}',
   1),

  ('std-fio-peak-write-iops',
   'peak-write-iops',
   'fio',
   '[global]
ioengine=libaio
direct=1
filename=<TARGET>
bs=4k
rw=randwrite
iodepth=32
numjobs=4
group_reporting=1
runtime=180
ramp_time=30
time_based=1
clat_percentiles=1
percentile_list=50:95:99:99.9
norandommap=1

[peak-write-iops]
',
   'Peak write IOPS characterization. 4K random 100 percent write, QD=32, 4 jobs. Saturates the write path. Not a realistic workload.',
   '{}',
   1),

  ('std-fio-peak-read-bw',
   'peak-read-bw',
   'fio',
   '[global]
ioengine=libaio
direct=1
filename=<TARGET>
bs=1M
rw=read
iodepth=16
numjobs=4
group_reporting=1
runtime=180
ramp_time=15
time_based=1
clat_percentiles=1
percentile_list=50:95:99:99.9

[peak-read-bw]
',
   'Peak read bandwidth characterization. 1M sequential 100 percent read, QD=16, 4 jobs. Maximum throughput exposed by the storage path under sequential reads.',
   '{}',
   1),

  ('std-fio-peak-write-bw',
   'peak-write-bw',
   'fio',
   '[global]
ioengine=libaio
direct=1
filename=<TARGET>
bs=1M
rw=write
iodepth=16
numjobs=4
group_reporting=1
runtime=180
ramp_time=15
time_based=1
clat_percentiles=1
percentile_list=50:95:99:99.9

[peak-write-bw]
',
   'Peak write bandwidth characterization. 1M sequential 100 percent write, QD=16, 4 jobs. Maximum write throughput exposed by the storage path under sequential writes.',
   '{}',
   1),

  ('std-fio-latency-read-qd1',
   'latency-read-qd1',
   'fio',
   '[global]
ioengine=libaio
direct=1
filename=<TARGET>
bs=4k
rw=randread
iodepth=1
numjobs=1
group_reporting=1
runtime=120
ramp_time=15
time_based=1
clat_percentiles=1
percentile_list=50:95:99:99.9
norandommap=1

[latency-read-qd1]
',
   'Read latency floor. 4K random read, QD=1, 1 job. Measures the per-IO round-trip time when the storage path is not queued. Lower is better; this is the responsiveness ceiling for synchronous client apps.',
   '{}',
   1),

  ('std-fio-latency-write-qd1',
   'latency-write-qd1',
   'fio',
   '[global]
ioengine=libaio
direct=1
filename=<TARGET>
bs=4k
rw=randwrite
iodepth=1
numjobs=1
group_reporting=1
runtime=120
ramp_time=15
time_based=1
clat_percentiles=1
percentile_list=50:95:99:99.9
norandommap=1

[latency-write-qd1]
',
   'Write latency floor. 4K random write, QD=1, 1 job. Measures the per-IO commit time including any journal or sync overhead. Lower is better.',
   '{}',
   1);
