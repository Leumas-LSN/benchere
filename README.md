<div align="center">

# Benchere

**Infrastructure benchmark toolkit for virtualization engineers.**

Provision ephemeral workers, run distributed storage and CPU benchmarks against a Proxmox cluster, deliver a presentable PDF report.

[![Latest release](https://img.shields.io/github/v/release/Leumas-LSN/benchere?style=flat-square&color=f97316)](https://github.com/Leumas-LSN/benchere/releases/latest)
[![Build status](https://img.shields.io/github/actions/workflow/status/Leumas-LSN/benchere/release.yml?style=flat-square)](https://github.com/Leumas-LSN/benchere/actions)
[![Go version](https://img.shields.io/badge/go-1.25-00ADD8?style=flat-square&logo=go&logoColor=white)](go.mod)
[![Vue 3](https://img.shields.io/badge/vue-3-42b883?style=flat-square&logo=vue.js&logoColor=white)](web/package.json)
[![License](https://img.shields.io/badge/license-MIT-555?style=flat-square)](LICENSE)

[Quick start](#quick-start) · [Features](#features) · [Architecture](#architecture) · [Build from source](#build-from-source) · [Roadmap](#roadmap)

</div>

---

## Why Benchere

Storage and CPU performance validation on a freshly delivered Proxmox cluster is traditionally a manual chore: SSH into a node, run a script, eyeball the numbers, transcribe them into a slide deck. Results are inconsistent, comparisons across clients are impossible, and there is no presentable artifact at the end.

Benchere standardizes the workflow:

- **One job** describes a target node, a worker shape, the storage pools to test, and the elbencho profiles to run.
- **One report** comes out — a dark-themed PDF with per-profile pass/fail verdicts based on configurable thresholds.
- **One binary** runs the whole stack: REST API, WebSocket live feed, embedded Vue 3 frontend, SQLite persistence.

## Quick start

The installer runs on any Debian 12+ or Ubuntu 22.04+ VM or LXC container with internet access. From the target machine, as root:

```bash
curl -fsSL https://github.com/Leumas-LSN/benchere/releases/latest/download/install.sh | sudo bash
```

Then open `http://<vm-ip>/` in a browser. The first visit launches an onboarding wizard that walks through:

1. **Language** — French or English (persisted, switchable later).
2. **Hypervisor** — Proxmox today; vSphere, Hyper-V and Azure Local listed as upcoming targets.
3. **Cluster connection** — API URL, token id/secret, deployment node, cluster identifier.
4. **Worker network** — bridge, static IP pool, CIDR, gateway.
5. **SSH key path** — used by Ansible to reach the worker VMs.

Once the wizard finishes, the dashboard is ready and the **New job** flow becomes available.

## Features

| Capability | Detail |
|---|---|
| Storage benchmarks | Distributed `elbencho` runs across N workers, IOPS read/write, throughput, latency p50 and p99. |
| CPU benchmarks | `stress-ng` over SSH on the workers, configurable stressors and timeouts. |
| Worker lifecycle | Cloud-init provisioning from a Debian generic image, static IP allocation from a user-defined pool, automatic teardown after the job completes. |
| Live monitoring | WebSocket feed, per-worker CPU / RAM / network / disk I/O updated every 2 seconds during the run. |
| Profile thresholds | Each elbencho profile carries optional pass/fail thresholds (min IOPS, max latency) that drive the verdict in the report. |
| Multi-pool runs | Pick several storage pools at job creation; one independent run per pool, named for unambiguous comparison. |
| PDF & HTML reports | Dark-themed by default, switchable to print mode. Localized to the user's chosen language. |
| Internationalization | Full FR / EN locales for the UI and the reports. |

## Architecture

```mermaid
flowchart LR
  user["Engineer"] -->|HTTPS| master["Master VM<br/>(this binary)"]
  master -->|REST + token| pve["Proxmox VE API"]
  master -->|SSH + key| workers["Worker VMs (N)"]
  master -->|persists| db[("SQLite<br/>jobs · results · profiles")]
  pve -->|cloud-init| workers
  workers -->|elbencho service<br/>(distributed)| workers
  workers -->|stress-ng| workers
  workers -->|live metrics<br/>via WS| user
```

Single Go binary, single deployable artifact:

```
cmd/benchere/        entry point
internal/
  api/               REST + WebSocket handlers
  proxmox/           Proxmox VE API client
  proxmoxhost/       (reserved for V2 host-side ops)
  ansible/           Ansible runner
  elbencho/          orchestration + live CSV parser
  stress/            stress-ng over SSH
  benchmark/         job orchestrator + IP allocator
  report/            HTML/PDF rendering + SVG charts
  ws/                WebSocket hub
  db/                SQLite migrations + queries
web/                 Vue 3 + Tailwind source (embedded via go:embed)
ansible/             worker provisioning playbooks
```

## Stack

- **Backend** — Go 1.25, Gorilla WebSocket, `modernc.org/sqlite` (pure Go, no CGO).
- **Frontend** — Vue 3 Composition API, Pinia, Vue Router, Tailwind CSS, Vite, vue-i18n.
- **Provisioning** — Ansible 2.x, Proxmox VE 8/9 REST API, cloud-init NoCloud.
- **Storage benchmark** — [elbencho](https://github.com/breuner/elbencho) in distributed mode (`--hosts`).
- **CPU benchmark** — [stress-ng](https://github.com/ColinIanKing/stress-ng) over SSH.
- **PDF rendering** — wkhtmltopdf.
- **CI** — GitHub Actions builds the Linux/amd64 binary on every tag, attaches it alongside `install.sh` to the release.

## Screenshots

Screenshots of the live dashboard, the job wizard and the PDF report are available in [`docs/screenshots/`](docs/screenshots/) (placeholder — populate during the next release cycle).

## Build from source

Prerequisites: Go 1.25+, Node.js 20+, npm, GNU Make.

```bash
git clone https://github.com/Leumas-LSN/benchere.git
cd benchere
make build       # builds web/dist via Vite, then the Go binary
make test
```

The binary embeds the frontend bundle via `//go:embed`. Any change in `web/src/` requires a Go rebuild to take effect at runtime.

To produce a versioned release locally:

```bash
make build VERSION=v1.7.0   # stamps main.Version via -ldflags
```

## Configuration

Runtime configuration is read from a few environment variables (defaults in parentheses):

| Variable | Default | Purpose |
|---|---|---|
| `BENCHERE_PORT` | `80` | HTTP listen port |
| `BENCHERE_DB` | `/opt/benchere/benchere.db` | SQLite database path |
| `BENCHERE_DEBUG` | `false` | Verbose request logging |
| `BENCHERE_SSH_KEY` | `/opt/benchere/id_rsa` | Private key Ansible uses to reach workers |

All other settings (Proxmox URL/token/node, storage, network bridge, IP pool, cluster name) live in the SQLite database and are managed through the onboarding wizard or the **Settings** page.

## Roadmap

V1 (current) targets Proxmox VE in an internal-network deployment without authentication. V2 will introduce:

- A `Hypervisor` interface to support **VMware vSphere**, **Microsoft Hyper-V** and **Azure Local** alongside Proxmox.
- An optional authentication layer for installs that face untrusted networks.
- A worker template builder to skip the cloud-image import on every run.
- Long-term metrics retention beyond the SQLite size limit (Prometheus or InfluxDB sink).

## License

MIT — see [LICENSE](LICENSE).
