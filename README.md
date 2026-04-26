# Benchere

> Storage and CPU benchmark tool for virtualization engineers — Proxmox-first, multi-hypervisor next.

[Français](#français) · [English](#english)

---

## English

Benchere measures the storage (via [elbencho](https://github.com/breuner/elbencho)) and CPU (via [stress-ng](https://github.com/ColinIanKing/stress-ng)) performance of a Proxmox cluster on demand, then turns the run into a PDF that you can hand to a client at a delivery meeting.

One job, a set of elbencho profiles, a report, a pass/fail verdict against thresholds.

### How it works

1. Connect Benchere to a Proxmox cluster (API URL + token).
2. Start a job: pick the node, storage pools, worker shape (vCPU/RAM/disks), and elbencho profiles.
3. Benchere provisions ephemeral Debian VMs via cloud-init, configures them with Ansible, runs the distributed benchmark, then tears the workers down.
4. Download the PDF report.

### Install

On any fresh Debian 12+ or Ubuntu 22.04+ VM (or LXC container) with internet access, run:

```bash
curl -fsSL https://github.com/Leumas-LSN/benchere/releases/latest/download/install.sh | sudo bash
```

Then open `http://<vm-ip>/` and follow the onboarding wizard. The wizard walks you through:

1. Language (FR/EN).
2. Hypervisor selection (Proxmox supported today; vSphere, Hyper-V, Azure Local listed for V2).
3. Cluster connection (API URL, token id/secret, deployment node, cluster identifier).
4. Worker network (bridge, static IP pool, CIDR, gateway).
5. SSH key path used by Ansible to reach the workers.

### Architecture

Single Go binary that serves the REST API, the WebSocket live feed and the embedded Vue3 frontend. SQLite for persistence. Ansible for worker provisioning.

```
cmd/benchere/        entry point
internal/
  api/               REST + WebSocket handlers
  proxmox/           Proxmox VE API client
  ansible/           Ansible runner
  elbencho/          orchestration + live CSV parser
  stress/            stress-ng over SSH
  benchmark/         state machine + job orchestrator + IP allocator
  report/            HTML/PDF rendering + SVG charts
  ws/                WebSocket hub
  db/                SQLite migrations + queries
web/                 Vue3 + Tailwind source
```

### Stack

- **Backend:** Go, SQLite (`modernc.org/sqlite`), Gorilla WebSocket
- **Frontend:** Vue3 (Composition API), Pinia, Vue Router, Tailwind CSS, Vite, vue-i18n
- **Provisioning:** Ansible, Proxmox VE API
- **Storage benchmark:** elbencho in distributed mode (`--hosts`)
- **CPU benchmark:** stress-ng over SSH
- **PDF rendering:** wkhtmltopdf

### Build from source

```bash
make build       # builds web/dist + the Go binary
make test
make clean
```

The binary embeds the frontend via `//go:embed`, so any change in `web/` requires a Go rebuild.

### Status

V1 — internal-network deployment only (no auth), Proxmox-only. V2 will introduce a `Hypervisor` interface for vSphere / Hyper-V / Azure Local.

---

## Français

Benchere mesure les performances stockage (via [elbencho](https://github.com/breuner/elbencho)) et CPU (via [stress-ng](https://github.com/ColinIanKing/stress-ng)) d'un cluster Proxmox à la demande, puis transforme le run en PDF présentable à un client en réunion de recette.

Un job, une liste de profils elbencho, un rapport, un verdict pass/fail face à des seuils.

### Comment ça marche

1. Tu connectes Benchere à un cluster Proxmox (URL API + token).
2. Tu lances un job : choix du node, des storage pools, dimensionnement des workers (vCPU/RAM/disques) et des profils elbencho.
3. Benchere provisionne des VMs Debian éphémères via cloud-init, les configure via Ansible, lance le benchmark en distribué, puis détruit les workers.
4. Tu télécharges le rapport PDF.

### Installation

Sur n'importe quelle VM Debian 12+ ou Ubuntu 22.04+ fraîche (ou un container LXC) avec accès internet :

```bash
curl -fsSL https://github.com/Leumas-LSN/benchere/releases/latest/download/install.sh | sudo bash
```

Ouvre ensuite `http://<ip-vm>/` et suis l'assistant d'onboarding. Il te demandera :

1. La langue (FR/EN).
2. Le choix de l'hyperviseur (Proxmox aujourd'hui ; vSphere, Hyper-V, Azure Local en V2).
3. Les paramètres de connexion au cluster (URL API, token id/secret, node, identifiant de cluster).
4. Le réseau workers (bridge, plage d'IPs statiques, CIDR, passerelle).
5. Le chemin de la clé SSH utilisée par Ansible.

### Architecture

Binaire Go unique qui sert l'API REST, le flux WebSocket temps réel et le frontend Vue3 embedé. SQLite pour la persistance. Ansible pour la configuration des workers.

```
cmd/benchere/        point d'entrée
internal/
  api/               handlers REST + WebSocket
  proxmox/           client API Proxmox VE
  ansible/           runner Ansible
  elbencho/          orchestration + parsing CSV live
  stress/            stress-ng via SSH
  benchmark/         state machine + orchestrateur de jobs + allocateur d'IPs
  report/            rendu HTML/PDF + charts SVG
  ws/                hub WebSocket
  db/                migrations + queries SQLite
web/                 source Vue3 + Tailwind
```

### Stack

- **Backend :** Go, SQLite (`modernc.org/sqlite`), Gorilla WebSocket
- **Frontend :** Vue3 (Composition API), Pinia, Vue Router, Tailwind CSS, Vite, vue-i18n
- **Provisioning :** Ansible, API Proxmox VE
- **Benchmark stockage :** elbencho en mode distribué (`--hosts`)
- **Benchmark CPU :** stress-ng via SSH
- **Rendu PDF :** wkhtmltopdf

### Build depuis les sources

```bash
make build       # build web/dist + binaire Go
make test
make clean
```

Le binaire embed le frontend via `//go:embed` : tout changement dans `web/` nécessite un rebuild Go.

### Statut

V1 — déploiement réseau interne uniquement (pas d'authentification), Proxmox seulement. V2 introduira une interface `Hypervisor` pour vSphere / Hyper-V / Azure Local.
