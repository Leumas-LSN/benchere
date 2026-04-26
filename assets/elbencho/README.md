# elbencho .deb

Place the elbencho Debian package here before running `make package`.

Expected filename: `elbencho_amd64.deb`

## Download

Get the latest release from the official repository:
https://github.com/breuner/elbencho/releases

Download the `elbencho_<version>_amd64.deb` package and rename it:

```bash
wget -O assets/elbencho/elbencho_amd64.deb \
  https://github.com/breuner/elbencho/releases/download/<version>/elbencho_<version>_amd64.deb
```

## Without elbencho

If you omit this file, the OVA build will succeed but the Master VM will not be
able to provision workers for storage benchmarks. The file will need to be placed
manually at `/opt/benchere/assets/elbencho_amd64.deb` on the Master VM before
running a storage job.
