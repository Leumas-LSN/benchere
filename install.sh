#!/bin/bash
# Benchere Master installer.
#
# Run inside a fresh Debian 12+ / Ubuntu 22.04+ VM or LXC container with
# internet access. Installs all dependencies, downloads the benchere
# binary and elbencho package, generates a per-instance SSH keypair, and
# starts the systemd service.
#
# Usage:
#
#   curl -fsSL https://github.com/Leumas-LSN/benchere/releases/latest/download/install.sh | sudo bash
#
# With options:
#
#   curl -fsSL https://github.com/Leumas-LSN/benchere/releases/latest/download/install.sh \
#     | sudo bash -s -- --version v1.0.2 --port 8080
#
# Flags:
#   --version <tag>   Benchere version to install (default: latest)
#   --port <num>      HTTP listen port (default: 80)
#   --no-elbencho     Skip elbencho download (storage benchmarks unavailable until provided manually)
#   --upgrade         Upgrade an existing install (replaces binary, restarts service, keeps DB and profiles)
#   --uninstall       Stop service and remove /opt/benchere (DB and profiles included)

set -euo pipefail

REPO="Leumas-LSN/benchere"
ELBENCHO_REPO="breuner/elbencho"

BENCHERE_VERSION="latest"
PORT="80"
NO_ELBENCHO="0"
UPGRADE="0"
UNINSTALL="0"

err()  { printf '\033[1;31merror:\033[0m %s\n' "$*" >&2; exit 1; }
info() { printf '\033[1;36m==>\033[0m %s\n' "$*"; }
ok()   { printf '\033[1;32m✓\033[0m %s\n' "$*"; }

while [ $# -gt 0 ]; do
  case "$1" in
    --version)     BENCHERE_VERSION="$2"; shift 2 ;;
    --port)        PORT="$2"; shift 2 ;;
    --no-elbencho) NO_ELBENCHO="1"; shift ;;
    --upgrade)     UPGRADE="1"; shift ;;
    --uninstall)   UNINSTALL="1"; shift ;;
    -h|--help)
      sed -n '2,/^set -e/p' "$0" | sed 's/^# \?//' | head -n -1
      exit 0
      ;;
    *) err "unknown flag: $1" ;;
  esac
done

# --- preflight ---
[ "$(id -u)" -eq 0 ] || err "must run as root (use sudo)"

if [ -r /etc/os-release ]; then
  . /etc/os-release
  case "$ID" in
    debian|ubuntu) : ;;
    *) err "unsupported OS '$ID'. Debian 12+ or Ubuntu 22.04+ required." ;;
  esac
else
  err "/etc/os-release not found, can't detect OS"
fi

# --- uninstall path ---
if [ "$UNINSTALL" = "1" ]; then
  info "stopping benchere service"
  systemctl stop benchere.service 2>/dev/null || true
  systemctl disable benchere.service 2>/dev/null || true
  systemctl disable benchere-firstboot.service 2>/dev/null || true
  rm -f /etc/systemd/system/benchere.service /etc/systemd/system/benchere-firstboot.service
  rm -f /etc/default/benchere
  systemctl daemon-reload
  rm -rf /opt/benchere
  ok "benchere removed."
  exit 0
fi

# --- resolve version ---
if [ "$BENCHERE_VERSION" = "latest" ]; then
  info "resolving latest release tag"
  BENCHERE_VERSION=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" \
            | grep -oE '"tag_name":\s*"[^"]+"' \
            | head -1 \
            | sed -E 's/.*"([^"]+)"/\1/')
  [ -n "$BENCHERE_VERSION" ] || err "could not determine latest release tag"
fi
info "installing benchere $BENCHERE_VERSION"

BINARY_URL="https://github.com/$REPO/releases/download/$BENCHERE_VERSION/benchere-linux-amd64"

# --- handle existing install ---
if [ -d /opt/benchere ] && [ "$UPGRADE" != "1" ]; then
  err "/opt/benchere already exists. Pass --upgrade to replace the binary in place, or --uninstall to wipe."
fi

# --- install OS deps ---
info "installing OS dependencies (apt)"
export DEBIAN_FRONTEND=noninteractive
apt-get update -qq
APT_PACKAGES=(ansible curl wget openssh-client ca-certificates python3)
# Headless Chromium is used by the report generator to produce PDFs.
# Debian ships it as 'chromium', Ubuntu as 'chromium-browser' (snap shim).
case "$ID" in
  debian) APT_PACKAGES+=(chromium) ;;
  ubuntu) APT_PACKAGES+=(chromium-browser) ;;
esac
# qemu-guest-agent is only meaningful in a real VM, harmless but useless in LXC.
# Install it only if we're in a VM (systemd-detect-virt reports kvm/qemu/vmware/etc, not 'lxc').
if command -v systemd-detect-virt >/dev/null 2>&1; then
  VIRT=$(systemd-detect-virt || true)
  case "$VIRT" in
    lxc|openvz|docker|none) : ;;
    *) APT_PACKAGES+=(qemu-guest-agent) ;;
  esac
fi
apt-get install -y -qq "${APT_PACKAGES[@]}" >/dev/null || \
  info "some packages did not install - PDF generation may be unavailable until a Chromium binary is on PATH"

# --- directory layout ---
mkdir -p /opt/benchere/ansible/playbooks
mkdir -p /opt/benchere/assets
mkdir -p /opt/benchere/profiles
mkdir -p /opt/benchere/output

# --- download binary ---
info "downloading benchere $BENCHERE_VERSION"
TMP=$(mktemp -d); trap 'rm -rf "$TMP"' EXIT
curl -fL --progress-bar -o "$TMP/benchere" "$BINARY_URL" \
  || err "binary download failed: $BINARY_URL"
install -m 0755 "$TMP/benchere" /opt/benchere/benchere

# --- ansible playbook (embedded inline so install.sh is self-contained) ---
cat > /opt/benchere/ansible/playbooks/provision_worker.yml <<'PLAYBOOK'
---
- name: Provision elbencho worker
  hosts: workers
  become: true
  vars:
    elbencho_deb: /tmp/elbencho_amd64.deb
  tasks:
    - name: Copy elbencho deb
      copy:
        src: "{{ elbencho_deb_local }}"
        dest: "{{ elbencho_deb }}"
        mode: '0644'
    - name: Install elbencho
      apt:
        deb: "{{ elbencho_deb }}"
        state: present
    - name: Install stress-ng and qemu-guest-agent
      apt:
        name:
          - stress-ng
          - qemu-guest-agent
        state: present
        update_cache: true
    - name: Start and enable qemu-guest-agent
      systemd:
        name: qemu-guest-agent
        state: started
        enabled: true
    - name: Create elbencho systemd service
      copy:
        dest: /etc/systemd/system/elbencho.service
        content: |
          [Unit]
          Description=Elbencho Benchmark Agent
          After=network.target
          [Service]
          Type=simple
          ExecStart=/usr/bin/elbencho --service --foreground
          Restart=always
          RestartSec=5
          User=root
          [Install]
          WantedBy=multi-user.target
    - name: Start and enable elbencho service
      systemd:
        name: elbencho
        state: started
        enabled: true
        daemon_reload: true
PLAYBOOK

# --- elbencho .deb (latest from upstream) ---
if [ "$NO_ELBENCHO" = "1" ]; then
  info "skipping elbencho (--no-elbencho)"
else
  info "downloading elbencho (latest from $ELBENCHO_REPO)"
  ELBENCHO_URL=$(curl -fsSL "https://api.github.com/repos/$ELBENCHO_REPO/releases/latest" \
                 | grep -oE '"browser_download_url":\s*"[^"]+_amd64\.deb"' \
                 | head -1 \
                 | sed -E 's/.*"([^"]+)"/\1/')
  if [ -n "$ELBENCHO_URL" ]; then
    curl -fL --progress-bar -o /opt/benchere/assets/elbencho_amd64.deb "$ELBENCHO_URL"
    info "installing elbencho on master"
    if ! dpkg -i /opt/benchere/assets/elbencho_amd64.deb >/dev/null 2>&1; then
      apt-get install -y -f -qq >/dev/null 2>&1
    fi
  else
    info "could not resolve elbencho release URL; storage benchmarks will be unavailable until you place /opt/benchere/assets/elbencho_amd64.deb manually"
  fi
fi

# --- per-instance SSH keypair (master → workers) ---
if [ ! -f /opt/benchere/id_rsa ]; then
  info "generating master SSH keypair"
  ssh-keygen -t ed25519 -N "" -f /opt/benchere/id_rsa -C "benchere-master-$(hostname)" >/dev/null
  chmod 600 /opt/benchere/id_rsa
  chmod 644 /opt/benchere/id_rsa.pub
fi

# --- systemd service ---
cat > /etc/default/benchere <<EOF
BENCHERE_PORT=$PORT
BENCHERE_DB=/opt/benchere/benchere.db
BENCHERE_DEBUG=false
EOF

cat > /etc/systemd/system/benchere.service <<'EOF'
[Unit]
Description=Benchere Infrastructure Benchmark Tool
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
EnvironmentFile=/etc/default/benchere
ExecStart=/opt/benchere/benchere
Restart=on-failure
RestartSec=5
User=root
WorkingDirectory=/opt/benchere
StandardOutput=journal
StandardError=journal
AmbientCapabilities=CAP_NET_BIND_SERVICE

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload

if [ "$UPGRADE" = "1" ]; then
  info "restarting benchere service (upgrade)"
  systemctl restart benchere.service
else
  info "enabling and starting benchere service"
  systemctl enable benchere.service >/dev/null
  systemctl start benchere.service
fi

# --- wait for HTTP up ---
info "waiting for HTTP on :$PORT"
DEADLINE=$(($(date +%s) + 60))
HTTP_OK=0
while [ "$(date +%s)" -lt "$DEADLINE" ]; do
  if curl -fsS --max-time 2 "http://127.0.0.1:$PORT/" >/dev/null 2>&1; then
    HTTP_OK=1
    break
  fi
  sleep 2
done

# --- summary ---
echo
if [ "$HTTP_OK" = "1" ]; then
  IP=$(hostname -I 2>/dev/null | awk '{print $1}')
  SUFFIX=""
  [ "$PORT" != "80" ] && SUFFIX=":$PORT"
  ok "benchere $BENCHERE_VERSION is up."
  echo "  open: http://${IP:-127.0.0.1}${SUFFIX}/"
  echo "  logs: journalctl -u benchere -f"
  echo "  next: configure Proxmox URL + API token in Settings."
else
  err "benchere is not responding on :$PORT yet. Check: journalctl -u benchere -n 50"
fi
