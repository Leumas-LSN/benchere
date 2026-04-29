-- Switch the 6 fio profiles from numjobs=8 iodepth=16 to numjobs=1 iodepth=128.
-- Multi-process fio (numjobs > 1) on a single block device filename without
-- size/offset_increment set has all processes reading/writing at offset 0
-- concurrently, biasing measurements. The HCIBench and Ceph upstream pattern
-- for block device benchmarking is a single process with a deeper queue.
-- Same total IOs in flight (128), simpler, more reproducible.

UPDATE benchmark_profiles
SET config_json = REPLACE(
    REPLACE(config_json, 'iodepth=16', 'iodepth=128'),
    'numjobs=8',
    'numjobs=1')
WHERE engine = 'fio';
