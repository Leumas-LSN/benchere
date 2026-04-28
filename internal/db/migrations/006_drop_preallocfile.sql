-- preallocfile=1 only has effect when elbencho writes to a regular file
-- (it triggers posix_fallocate). Benchere drives elbencho against block
-- devices (/dev/disk/by-id/scsi-...) where the option is a no-op. Drop it
-- from every profile config so the rendered Profils testes section in the
-- report does not advertise something elbencho silently ignores.
--
-- The seed in migration 004 contains CRLF line endings, so strip both the
-- CRLF and LF variants.
UPDATE benchmark_profiles
SET config_json = REPLACE(
    REPLACE(config_json,
        'preallocfile=1' || CHAR(13) || CHAR(10),
        ''),
    'preallocfile=1' || CHAR(10),
    '');
