#!/usr/bin/env bash
#
# RPM Manager — One-line installer
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/yeagoo/rpmmanager/main/install.sh | bash
#   curl -fsSL https://raw.githubusercontent.com/yeagoo/rpmmanager/main/install.sh | bash -s -- --version v0.1.0
#
# Options (via environment or flags):
#   --version VERSION   Specific version to install (default: latest)
#   --prefix  PATH      Install prefix (default: /usr/local/bin)
#   --no-service        Skip systemd service setup
#
set -euo pipefail

# ── Defaults ──────────────────────────────────────────────────────
REPO="yeagoo/rpmmanager"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/rpmmanager"
DATA_DIR="/var/lib/rpmmanager"
LOG_DIR="/var/log/rpmmanager"
SERVICE_USER="rpmmanager"
VERSION=""
SKIP_SERVICE=false

# ── Color helpers ─────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m'

info()  { echo -e "${BLUE}[INFO]${NC}  $*"; }
ok()    { echo -e "${GREEN}[OK]${NC}    $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC}  $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*" >&2; exit 1; }

# ── Parse args ────────────────────────────────────────────────────
while [[ $# -gt 0 ]]; do
    case "$1" in
        --version)    VERSION="$2"; shift 2 ;;
        --prefix)     INSTALL_DIR="$2"; shift 2 ;;
        --no-service) SKIP_SERVICE=true; shift ;;
        *)            error "Unknown option: $1" ;;
    esac
done

# ── Detect platform ──────────────────────────────────────────────
detect_platform() {
    local os arch

    os="$(uname -s | tr '[:upper:]' '[:lower:]')"
    case "$os" in
        linux)  os="linux" ;;
        *)      error "Unsupported OS: $os (only Linux is supported)" ;;
    esac

    arch="$(uname -m)"
    case "$arch" in
        x86_64|amd64)   arch="amd64" ;;
        aarch64|arm64)  arch="arm64" ;;
        *)              error "Unsupported architecture: $arch" ;;
    esac

    echo "${os}-${arch}"
}

# ── Get latest version ───────────────────────────────────────────
get_latest_version() {
    local url="https://api.github.com/repos/${REPO}/releases/latest"
    local tag

    if command -v curl &>/dev/null; then
        tag=$(curl -fsSL "$url" 2>/dev/null | grep '"tag_name"' | head -1 | cut -d'"' -f4)
    elif command -v wget &>/dev/null; then
        tag=$(wget -qO- "$url" 2>/dev/null | grep '"tag_name"' | head -1 | cut -d'"' -f4)
    else
        error "Neither curl nor wget found. Please install one of them."
    fi

    if [[ -z "$tag" ]]; then
        error "Failed to fetch latest version. Specify manually with --version"
    fi
    echo "$tag"
}

# ── Download file ─────────────────────────────────────────────────
download() {
    local url="$1" dest="$2"
    if command -v curl &>/dev/null; then
        curl -fsSL -o "$dest" "$url"
    elif command -v wget &>/dev/null; then
        wget -qO "$dest" "$url"
    fi
}

# ── Check root ────────────────────────────────────────────────────
need_root() {
    if [[ $EUID -ne 0 ]]; then
        error "This script must be run as root (or with sudo)"
    fi
}

# ── Main ──────────────────────────────────────────────────────────
main() {
    echo ""
    echo -e "${BLUE}╔══════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║       RPM Manager Installer          ║${NC}"
    echo -e "${BLUE}╚══════════════════════════════════════╝${NC}"
    echo ""

    need_root

    # Detect platform
    local platform
    platform=$(detect_platform)
    info "Platform: ${platform}"

    # Get version
    if [[ -z "$VERSION" ]]; then
        info "Fetching latest version..."
        VERSION=$(get_latest_version)
    fi
    info "Version:  ${VERSION}"

    # Download binary
    local binary_name="rpmmanager-${platform}"
    local download_url="https://github.com/${REPO}/releases/download/${VERSION}/${binary_name}"
    local tmp_dir
    tmp_dir=$(mktemp -d)
    trap "rm -rf ${tmp_dir}" EXIT

    info "Downloading ${binary_name}..."
    download "$download_url" "${tmp_dir}/rpmmanager" || error "Download failed. Check the version and try again."

    # Download checksums and verify
    local checksum_url="https://github.com/${REPO}/releases/download/${VERSION}/checksums.txt"
    if download "$checksum_url" "${tmp_dir}/checksums.txt" 2>/dev/null; then
        info "Verifying checksum..."
        local expected actual
        expected=$(grep "${binary_name}" "${tmp_dir}/checksums.txt" | awk '{print $1}')
        actual=$(sha256sum "${tmp_dir}/rpmmanager" | awk '{print $1}')
        if [[ "$expected" != "$actual" ]]; then
            error "Checksum mismatch! Expected: ${expected}, Got: ${actual}"
        fi
        ok "Checksum verified"
    else
        warn "Could not download checksums, skipping verification"
    fi

    # Install binary
    info "Installing to ${INSTALL_DIR}/rpmmanager..."
    install -m 755 "${tmp_dir}/rpmmanager" "${INSTALL_DIR}/rpmmanager"
    ok "Binary installed: ${INSTALL_DIR}/rpmmanager"

    # Verify
    local installed_version
    installed_version=$("${INSTALL_DIR}/rpmmanager" version 2>/dev/null || echo "unknown")
    ok "Installed: ${installed_version}"

    # Create user
    if ! id "$SERVICE_USER" &>/dev/null; then
        info "Creating system user: ${SERVICE_USER}"
        useradd -r -s /sbin/nologin -d "$DATA_DIR" "$SERVICE_USER"
        ok "User created"
    fi

    # Create directories
    info "Creating directories..."
    mkdir -p "$CONFIG_DIR" "$DATA_DIR"/{repos,logs,tmp,gnupg} "$LOG_DIR"
    chown -R "${SERVICE_USER}:${SERVICE_USER}" "$DATA_DIR" "$LOG_DIR"
    chmod 700 "$DATA_DIR/gnupg"
    ok "Directories created"

    # Create default config if not exists
    if [[ ! -f "${CONFIG_DIR}/config.yaml" ]]; then
        info "Creating default config..."
        cat > "${CONFIG_DIR}/config.yaml" <<'YAML'
server:
  listen: "127.0.0.1:8080"
  base_url: "http://localhost:8080"

auth:
  username: "admin"
  # password_hash, jwt_secret, api_token are auto-generated on first run

database:
  path: "/var/lib/rpmmanager/rpmmanager.db"

storage:
  repo_root: "/var/lib/rpmmanager/repos"
  build_logs: "/var/lib/rpmmanager/logs"
  temp_dir: "/var/lib/rpmmanager/tmp"

gpg:
  home_dir: "/var/lib/rpmmanager/gnupg"

monitor:
  enabled: true
  default_interval: "6h"

log:
  level: "info"
  format: "text"
YAML
        chown root:${SERVICE_USER} "${CONFIG_DIR}/config.yaml"
        chmod 640 "${CONFIG_DIR}/config.yaml"
        ok "Config created: ${CONFIG_DIR}/config.yaml"
    else
        warn "Config already exists, skipping: ${CONFIG_DIR}/config.yaml"
    fi

    # Systemd service
    if [[ "$SKIP_SERVICE" == false ]] && command -v systemctl &>/dev/null; then
        info "Installing systemd service..."
        cat > /etc/systemd/system/rpmmanager.service <<EOF
[Unit]
Description=RPM Manager
After=network.target

[Service]
Type=simple
User=${SERVICE_USER}
Group=${SERVICE_USER}
ExecStart=${INSTALL_DIR}/rpmmanager serve --config ${CONFIG_DIR}/config.yaml
Restart=on-failure
RestartSec=5
LimitNOFILE=65535

NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=${DATA_DIR} ${LOG_DIR}
PrivateTmp=true

[Install]
WantedBy=multi-user.target
EOF
        systemctl daemon-reload
        ok "Systemd service installed"

        echo ""
        info "To start RPM Manager:"
        echo "  systemctl enable --now rpmmanager"
        echo ""
        info "To check status:"
        echo "  systemctl status rpmmanager"
        echo "  journalctl -u rpmmanager -f"
    elif [[ "$SKIP_SERVICE" == true ]]; then
        info "Skipping systemd service (--no-service)"
    else
        warn "systemctl not found, skipping service installation"
    fi

    # Check runtime dependencies
    echo ""
    info "Checking runtime dependencies..."
    local missing=()
    command -v createrepo_c &>/dev/null || missing+=("createrepo_c")
    command -v gpg &>/dev/null         || missing+=("gnupg2")
    command -v rpmsign &>/dev/null     || missing+=("rpm-sign")
    command -v nfpm &>/dev/null        || missing+=("nfpm")

    if [[ ${#missing[@]} -gt 0 ]]; then
        warn "Missing optional dependencies: ${missing[*]}"
        echo ""
        echo "  Install on RHEL/AlmaLinux/Rocky:"
        echo "    dnf install createrepo_c gnupg2 rpm-sign rpmlint"
        echo ""
        echo "  Install nfpm:"
        echo "    rpm -i https://github.com/goreleaser/nfpm/releases/download/v2.41.1/nfpm_2.41.1_\$(uname -m | sed 's/x86_64/x86_64/;s/aarch64/aarch64/').rpm"
    else
        ok "All runtime dependencies found"
    fi

    # Done
    echo ""
    echo -e "${GREEN}╔══════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║     Installation complete!           ║${NC}"
    echo -e "${GREEN}╚══════════════════════════════════════╝${NC}"
    echo ""
    echo "  Binary:  ${INSTALL_DIR}/rpmmanager"
    echo "  Config:  ${CONFIG_DIR}/config.yaml"
    echo "  Data:    ${DATA_DIR}/"
    echo ""
    echo "  First run will print the generated admin password to stderr."
    echo "  Make sure to save it!"
    echo ""
}

main "$@"
