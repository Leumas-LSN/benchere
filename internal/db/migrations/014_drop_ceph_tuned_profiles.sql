-- Migration 014: remove the ceph-tuned fio profiles seeded in migration 010.
--
-- Product intent: the SAME profiles must run unchanged across every block
-- storage backend (iSCSI, NFS, DataCore, vSAN, Ceph RBD, LVM thin, local
-- NVMe, etc.) so customers can compare results apples to apples. Backend-
-- specific tuning contradicts that positioning. The six industry-standard
-- profiles seeded by migration 012 (oltp-4k-70-30, sql-8k-70-30,
-- vdi-4k-20-80, backup-256k-read, backup-256k-write, mixed-32k-50-50)
-- stay; everything ceph-prefixed is dropped.
--
-- Historical jobs that ran a ceph-* profile keep their results rows. The
-- profile definition is gone so the JobDetailView verdict tooltip will
-- show N/A for those rows. Acceptable; data is not lost.

DELETE FROM benchmark_profiles WHERE engine = 'fio' AND name LIKE 'ceph-%';
