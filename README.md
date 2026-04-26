# Benchere

Outil de benchmark d'infrastructure pour ingénieurs en virtualisation. Mesure objectivement les performances stockage (via [elbencho](https://github.com/breuner/elbencho)) et CPU (via [stress-ng](https://github.com/ColinIanKing/stress-ng)) d'une infrastructure Proxmox, et produit un rapport PDF présentable à un client.

Un job, des profils, un rapport, un verdict pass/fail face à des seuils.

## Comment ça marche

1. Tu connectes Benchere à un cluster Proxmox (URL + token API).
2. Tu lances un job : choix du nœud, du storage pool, du nombre de workers, et des profils elbencho à exécuter.
3. Benchere provisionne à la demande des VMs Debian 12 éphémères (cloud-init), les configure via Ansible, lance les benchmarks en distribué, puis détruit les workers à la fin.
4. Tu télécharges le rapport PDF, et tu le poses sur la table.

## Architecture

Monolithe Go : un seul binaire sert l'API REST, le WebSocket temps réel et le frontend Vue3 compilé statiquement. SQLite pour la persistance. Ansible pour le provisioning des workers.

```
cmd/benchere/        entry point
internal/
  api/               handlers REST + WebSocket
  proxmox/           client API Proxmox VE
  ansible/           runner Ansible
  elbencho/          orchestration + parsing live CSV
  stress/            stress-ng via SSH
  benchmark/         state machine + orchestrator des jobs
  report/            génération PDF/HTML + SVG charts
  ws/                WebSocket hub
  db/                SQLite migrations + queries
web/                 Vue3 + Tailwind (source)
ansible/             playbooks provisioning workers
packer/              build OVA Master
```

## Stack

- **Backend** : Go, SQLite (`modernc.org/sqlite`), Gorilla WebSocket
- **Frontend** : Vue3 (Composition API), Pinia, Vue Router, Tailwind CSS, Vite
- **Provisioning** : Ansible, API Proxmox VE
- **Benchmark stockage** : elbencho en mode distribué (`--hosts`)
- **Benchmark CPU** : stress-ng sur workers via SSH
- **Build OVA** : Packer + Debian 12 cloud image

## Build

```bash
make build       # frontend (npm run build) puis go build → ./benchere
make package     # OVA via Packer
make clean
```

Le binaire embed `web/dist/` via `//go:embed`. Tout changement frontend requiert un rebuild Go.

## Variables d'environnement

| Variable          | Défaut             | Rôle                          |
|-------------------|--------------------|-------------------------------|
| `BENCHERE_PORT`   | `80`               | Port d'écoute HTTP            |
| `BENCHERE_DB`     | `/data/benchere.db`| Chemin SQLite                 |
| `BENCHERE_DEBUG`  | `false`            | Logs verbeux                  |

## Concepts

- **Job** : unité de benchmark. Modes `storage` / `cpu` / `mixed`. États `pending → provisioning → running → done | failed | cancelled`.
- **Worker** : VM Debian 12 éphémère créée dans Proxmox via cloud-init, provisionnée par Ansible, détruite à la fin du job.
- **Profil elbencho** : nom + config (block size, ratio R/W, pattern) + thresholds (`min_iops_read`, `min_iops_write`, `max_latency_ms`) qui produisent le verdict pass/fail.

## Statut

V1 — déploiement réseau interne uniquement, pas d'authentification, hyperviseur Proxmox uniquement. Multi-hyperviseur prévu en V2 (interface `Hypervisor` dans `internal/proxmox`).
