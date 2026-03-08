#!/usr/bin/env bash
#
# RPM Manager — One-line installer
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/yeagoo/rpmmanager/main/install.sh | bash
#   curl -fsSL https://raw.githubusercontent.com/yeagoo/rpmmanager/main/install.sh | bash -s -- --version v0.1.0
#
# Options:
#   --version    VERSION   Specific version to install (default: latest)
#   --prefix     PATH      Install prefix (default: /usr/local/bin)
#   --domain     DOMAIN    Public repo domain (enables Caddy setup)
#   --admin      DOMAIN    Admin panel domain (default: admin.<domain>)
#   --admin-port PORT      Admin panel HTTPS port (default: 28088)
#   --no-service           Skip systemd service setup
#   --no-caddy             Skip Caddy setup prompt
#   --yes                  Non-interactive mode, answer yes to all prompts
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
REPO_DOMAIN=""
ADMIN_DOMAIN=""
ADMIN_PORT="28088"
SKIP_SERVICE=false
SKIP_CADDY=false
AUTO_YES=false

# ── Color helpers ─────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
BOLD='\033[1m'
NC='\033[0m'

info()  { echo -e "${BLUE}[INFO]${NC}  $*"; }
ok()    { echo -e "${GREEN}[OK]${NC}    $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC}  $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*" >&2; exit 1; }

ask() {
    if [[ "$AUTO_YES" == true ]]; then return 0; fi
    local prompt="$1 [y/N] "
    local answer
    echo -en "${BOLD}${prompt}${NC}"
    read -r answer </dev/tty
    [[ "$answer" =~ ^[Yy]$ ]]
}

ask_input() {
    local prompt="$1" default="$2"
    local answer
    echo -en "${BOLD}${prompt} [${default}]: ${NC}" >&2
    read -r answer </dev/tty
    echo "${answer:-$default}"
}

# ── Parse args ────────────────────────────────────────────────────
while [[ $# -gt 0 ]]; do
    case "$1" in
        --version)    VERSION="$2"; shift 2 ;;
        --prefix)     INSTALL_DIR="$2"; shift 2 ;;
        --domain)     REPO_DOMAIN="$2"; shift 2 ;;
        --admin)      ADMIN_DOMAIN="$2"; shift 2 ;;
        --admin-port) ADMIN_PORT="$2"; shift 2 ;;
        --no-service) SKIP_SERVICE=true; shift ;;
        --no-caddy)   SKIP_CADDY=true; shift ;;
        --yes|-y)     AUTO_YES=true; shift ;;
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

download() {
    local url="$1" dest="$2"
    if command -v curl &>/dev/null; then
        curl -fsSL -o "$dest" "$url"
    elif command -v wget &>/dev/null; then
        wget -qO "$dest" "$url"
    fi
}

need_root() {
    if [[ $EUID -ne 0 ]]; then
        error "This script must be run as root (or with sudo)"
    fi
}

detect_pkg_manager() {
    if command -v dnf &>/dev/null; then
        echo "dnf"
    elif command -v yum &>/dev/null; then
        echo "yum"
    elif command -v apt-get &>/dev/null; then
        echo "apt"
    else
        echo "unknown"
    fi
}

# ── Install runtime dependencies ─────────────────────────────────
install_dependencies() {
    local pkg_mgr
    pkg_mgr=$(detect_pkg_manager)

    info "Installing runtime dependencies..."

    case "$pkg_mgr" in
        dnf|yum)
            $pkg_mgr install -y createrepo_c gnupg2 rpm-sign rpmlint 2>/dev/null || true
            if ! command -v nfpm &>/dev/null; then
                info "Installing nfpm..."
                local nfpm_arch nfpm_ver
                nfpm_arch=$(uname -m)
                nfpm_ver=$(curl -fsSL https://api.github.com/repos/goreleaser/nfpm/releases/latest 2>/dev/null | grep '"tag_name"' | head -1 | cut -d'"' -f4 | sed 's/^v//')
                nfpm_ver="${nfpm_ver:-2.45.0}"
                rpm -i "https://github.com/goreleaser/nfpm/releases/download/v${nfpm_ver}/nfpm-${nfpm_ver}-1.${nfpm_arch}.rpm" 2>/dev/null || \
                    warn "nfpm installation failed — install manually: https://nfpm.goreleaser.com/install/"
            fi
            ;;
        apt)
            apt-get update -qq
            apt-get install -y -qq createrepo-c gnupg2 rpm rpmlint 2>/dev/null || true
            ;;
        *)
            warn "Unknown package manager, skipping dependency installation"
            warn "Please manually install: createrepo_c gnupg2 rpm-sign nfpm"
            ;;
    esac
}

# ── Install Caddy ─────────────────────────────────────────────────
install_caddy() {
    if command -v caddy &>/dev/null; then
        ok "Caddy already installed: $(caddy version 2>/dev/null | head -1)"
        return 0
    fi

    local pkg_mgr
    pkg_mgr=$(detect_pkg_manager)

    info "Installing Caddy..."

    case "$pkg_mgr" in
        dnf|yum)
            $pkg_mgr install -y 'dnf-command(copr)' 2>/dev/null || true
            $pkg_mgr copr enable -y @caddy/caddy 2>/dev/null || true
            $pkg_mgr install -y caddy || {
                warn "Caddy repo install failed, trying direct download..."
                install_caddy_binary
                return $?
            }
            ;;
        apt)
            apt-get install -y -qq debian-keyring debian-archive-keyring apt-transport-https 2>/dev/null
            curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg 2>/dev/null
            curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | tee /etc/apt/sources.list.d/caddy-stable.list >/dev/null
            apt-get update -qq
            apt-get install -y -qq caddy || {
                warn "Caddy apt install failed, trying direct download..."
                install_caddy_binary
                return $?
            }
            ;;
        *)
            install_caddy_binary
            return $?
            ;;
    esac

    ok "Caddy installed"
}

install_caddy_binary() {
    local arch
    arch=$(uname -m)
    case "$arch" in
        x86_64)  arch="amd64" ;;
        aarch64) arch="arm64" ;;
    esac

    local caddy_url="https://caddyserver.com/api/download?os=linux&arch=${arch}"
    info "Downloading Caddy binary..."
    download "$caddy_url" "/usr/local/bin/caddy" || {
        error "Failed to download Caddy"
    }
    chmod +x /usr/local/bin/caddy

    if ! id caddy &>/dev/null; then
        useradd -r -s /sbin/nologin -d /var/lib/caddy caddy
    fi
    mkdir -p /var/lib/caddy /var/log/caddy /etc/caddy
    chown caddy:caddy /var/lib/caddy /var/log/caddy

    if [[ ! -f /etc/systemd/system/caddy.service ]]; then
        cat > /etc/systemd/system/caddy.service <<'CSVC'
[Unit]
Description=Caddy
After=network.target network-online.target
Requires=network-online.target

[Service]
Type=notify
User=caddy
Group=caddy
ExecStart=/usr/local/bin/caddy run --environ --config /etc/caddy/Caddyfile
ExecReload=/usr/local/bin/caddy reload --config /etc/caddy/Caddyfile
TimeoutStopSec=5s
LimitNOFILE=1048576

[Install]
WantedBy=multi-user.target
CSVC
        systemctl daemon-reload
    fi
    ok "Caddy binary installed"
}

# ── Configure firewall ───────────────────────────────────────────
configure_firewall() {
    local admin_port="$1"
    echo ""

    if command -v firewall-cmd &>/dev/null; then
        info "Configuring firewalld..."
        firewall-cmd --permanent --add-service=http --add-service=https &>/dev/null || true
        firewall-cmd --permanent --add-port="${admin_port}/tcp" &>/dev/null || true
        firewall-cmd --reload &>/dev/null || true
        ok "Firewall ports opened: 80, 443, ${admin_port}"
    elif command -v ufw &>/dev/null; then
        info "Configuring ufw..."
        ufw allow 80/tcp &>/dev/null || true
        ufw allow 443/tcp &>/dev/null || true
        ufw allow "${admin_port}/tcp" &>/dev/null || true
        ok "Firewall ports opened: 80, 443, ${admin_port}"
    elif command -v iptables &>/dev/null && iptables -L INPUT -n &>/dev/null 2>&1; then
        info "No firewalld/ufw detected. If you have a firewall, open these ports manually:"
        echo "    80, 443    — Public repo (Caddy auto-HTTPS)"
        echo "    ${admin_port}       — Admin panel"
    else
        info "No firewall detected, skipping"
    fi
}

# ── Verify Caddy TLS certificates ───────────────────────────────
verify_caddy_tls() {
    local url="$1"
    local max_attempts=3
    local wait_secs=5

    echo ""
    info "Verifying TLS certificate for ${url} ..."
    sleep "${wait_secs}"

    for attempt in $(seq 1 "$max_attempts"); do
        if curl -sf --connect-timeout 5 --max-time 10 -o /dev/null "${url}" 2>/dev/null; then
            ok "TLS certificate verified"
            return 0
        fi
        if [[ "$attempt" -lt "$max_attempts" ]]; then
            warn "Attempt ${attempt}/${max_attempts} failed, restarting Caddy and retrying..."
            systemctl restart caddy
            sleep "${wait_secs}"
        fi
    done

    warn "TLS certificate not ready yet. This may be a transient ACME issue."
    info "Try manually: systemctl restart caddy"
    info "Check logs:   journalctl -u caddy | grep -i cert"
    return 0
}

# ── Configure Caddy (two-domain architecture) ────────────────────
configure_caddy() {
    local repo_domain="$1"
    local admin_domain="$2"
    local admin_port="$3"
    local caddyfile="/etc/caddy/Caddyfile"

    mkdir -p /etc/caddy /var/log/caddy
    chown caddy:caddy /var/log/caddy
    if [[ -f "$caddyfile" ]]; then
        local backup="${caddyfile}.bak.$(date +%s)"
        warn "Existing Caddyfile found, backing up to ${backup}"
        cp "$caddyfile" "$backup"
    fi

    info "Generating Caddyfile..."
    info "  Repo:  ${repo_domain} (:80/:443)"
    info "  Admin: ${admin_domain} (:${admin_port})"

    cat > "$caddyfile" <<EOF
# RPM Manager — auto-generated by install.sh
#
# Public repo:  ${repo_domain} (ports 80/443)
# Admin panel:  ${admin_domain}:${admin_port} (HTTPS)

# ── Public RPM Repository ────────────────────────────────────────
# Serves RPM packages, repodata, GPG keys directly from filesystem.
# No authentication required — this is what end users access.
#
${repo_domain} {
	root * ${DATA_DIR}/repos

	file_server {
		browse
	}

	# Hide internal directories
	@hidden path /.rollback/* /*/templates/*
	respond @hidden 404

	# Repo RPM public download (e.g. dnf install https://${repo_domain}/caddy/repo-rpm/caddy-repo-1.0-1.noarch.rpm)
	# These are served by rpmmanager backend
	@reporpm path_regexp reporpm ^/[^/]+/repo-rpm/.+
	handle @reporpm {
		reverse_proxy localhost:8080
	}

	# ── Cache headers ────────────────────────────────────────
	@rpm path *.rpm
	header @rpm Cache-Control "public, max-age=86400, immutable"

	@repodata path */repodata/*
	header @repodata Cache-Control "public, max-age=300"

	@gpgkey path */gpg.key
	header @gpgkey Cache-Control "public, max-age=604800"

	@repofile path *.repo
	header @repofile Cache-Control "public, max-age=3600"

	# Security headers
	header {
		X-Content-Type-Options "nosniff"
		X-Frame-Options "DENY"
		-Server
	}

	log {
		output file /var/log/caddy/repo-access.log {
			roll_size 100MiB
			roll_keep 5
		}
	}
}

# ── Admin Panel ──────────────────────────────────────────────────
# Management UI and API. Runs on a separate domain and custom port.
# Consider restricting access via firewall or IP whitelist.
#
${admin_domain}:${admin_port} {
	reverse_proxy localhost:8080

	header {
		X-Content-Type-Options "nosniff"
		X-Frame-Options "SAMEORIGIN"
		-Server
	}

	log {
		output file /var/log/caddy/admin-access.log {
			roll_size 50MiB
			roll_keep 3
		}
	}
}
EOF

    # Grant caddy user read access to repo data
    usermod -aG "${SERVICE_USER}" caddy 2>/dev/null || true
    chmod 750 "${DATA_DIR}"
    chmod 755 "${DATA_DIR}/repos"

    ok "Caddyfile generated: ${caddyfile}"
}

# ══════════════════════════════════════════════════════════════════
# ── Main ─────────────────────────────────────────────────────────
# ══════════════════════════════════════════════════════════════════
main() {
    echo ""
    echo -e "${BLUE}╔══════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║       RPM Manager Installer          ║${NC}"
    echo -e "${BLUE}╚══════════════════════════════════════╝${NC}"
    echo ""

    need_root

    # ── Step 1: Detect platform ──────────────────────────────
    local platform
    platform=$(detect_platform)
    info "Platform: ${platform}"

    # ── Step 2: Get version ──────────────────────────────────
    if [[ -z "$VERSION" ]]; then
        info "Fetching latest version..."
        VERSION=$(get_latest_version)
    fi
    info "Version:  ${VERSION}"

    # ── Step 3: Download & verify binary ─────────────────────
    local binary_name="rpmmanager-${platform}"
    local download_url="https://github.com/${REPO}/releases/download/${VERSION}/${binary_name}"
    local tmp_dir
    tmp_dir=$(mktemp -d)
    trap "rm -rf ${tmp_dir}" EXIT

    info "Downloading ${binary_name}..."
    download "$download_url" "${tmp_dir}/rpmmanager" || error "Download failed. Check the version and try again."

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

    # ── Step 4: Install binary ───────────────────────────────
    info "Installing to ${INSTALL_DIR}/rpmmanager..."
    install -m 755 "${tmp_dir}/rpmmanager" "${INSTALL_DIR}/rpmmanager"
    ok "Binary installed: ${INSTALL_DIR}/rpmmanager"

    local installed_version
    installed_version=$("${INSTALL_DIR}/rpmmanager" version 2>/dev/null || echo "unknown")
    ok "Installed: ${installed_version}"

    # ── Step 5: Create user & directories ────────────────────
    if ! id "$SERVICE_USER" &>/dev/null; then
        info "Creating system user: ${SERVICE_USER}"
        useradd -r -s /sbin/nologin -d "$DATA_DIR" "$SERVICE_USER"
        ok "User created"
    fi

    info "Creating directories..."
    mkdir -p "$CONFIG_DIR" "$DATA_DIR"/{repos,logs,tmp,gnupg} "$LOG_DIR"
    chown -R "${SERVICE_USER}:${SERVICE_USER}" "$DATA_DIR" "$LOG_DIR"
    chmod 700 "$DATA_DIR/gnupg"
    ok "Directories created"

    # ── Step 6: Install runtime dependencies ─────────────────
    if ask "Install runtime dependencies (createrepo_c, gnupg2, rpm-sign, nfpm)?"; then
        install_dependencies
    else
        info "Skipping dependency installation"
    fi

    # ── Step 7: Default config ───────────────────────────────
    if [[ ! -f "${CONFIG_DIR}/config.yaml" ]]; then
        info "Creating default config..."
        cat > "${CONFIG_DIR}/config.yaml" <<YAML
server:
  listen: "127.0.0.1:8080"
  base_url: "http://localhost:8080"

auth:
  username: "admin"
  password_hash: ""
  jwt_secret: ""
  api_token: ""

database:
  path: "${DATA_DIR}/rpmmanager.db"

storage:
  repo_root: "${DATA_DIR}/repos"
  build_logs: "${DATA_DIR}/logs"
  temp_dir: "${DATA_DIR}/tmp"

gpg:
  home_dir: "${DATA_DIR}/gnupg"

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

    # ── Step 8: Systemd service for rpmmanager ───────────────
    if [[ "$SKIP_SERVICE" == false ]] && command -v systemctl &>/dev/null; then
        info "Installing rpmmanager systemd service..."
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
        ok "rpmmanager.service installed"
    fi

    # ── Step 9: Caddy setup ──────────────────────────────────
    local setup_caddy=false

    if [[ "$SKIP_CADDY" == false ]] && [[ -n "$REPO_DOMAIN" ]]; then
        # Domains provided via flags
        if [[ -z "$ADMIN_DOMAIN" ]]; then
            ADMIN_DOMAIN="admin.${REPO_DOMAIN}"
        fi
        setup_caddy=true
    elif [[ "$SKIP_CADDY" == false ]] && [[ -z "$REPO_DOMAIN" ]]; then
        echo ""
        echo -e "${BOLD}── Caddy Setup ──${NC}"
        echo ""
        echo "  RPM Manager uses a two-domain architecture:"
        echo ""
        echo "    rpms.example.com        → Public RPM repo (ports 80/443)"
        echo "       Direct file serving for packages, repodata, GPG keys"
        echo ""
        echo "    admin.rpms.example.com  → Admin panel (custom port, default ${ADMIN_PORT})"
        echo "       Management UI, build triggers, settings"
        echo ""
        echo "  Both are served by Caddy with automatic HTTPS (Let's Encrypt)."
        echo ""

        if ask "Set up Caddy with this two-domain architecture?"; then
            REPO_DOMAIN=$(ask_input "Public repo domain" "rpms.example.com")
            local default_admin="admin.${REPO_DOMAIN}"
            ADMIN_DOMAIN=$(ask_input "Admin panel domain" "${default_admin}")
            ADMIN_PORT=$(ask_input "Admin panel HTTPS port" "${ADMIN_PORT}")
            setup_caddy=true
        fi
    fi

    if [[ "$setup_caddy" == true ]]; then
        install_caddy
        configure_caddy "$REPO_DOMAIN" "$ADMIN_DOMAIN" "$ADMIN_PORT"

        # Update rpmmanager config with the real URLs
        if [[ -f "${CONFIG_DIR}/config.yaml" ]]; then
            # base_url = admin panel URL (used for CORS and internal links)
            local admin_url="https://${ADMIN_DOMAIN}:${ADMIN_PORT}"
            # repo_base_url = public repo URL (used in .repo files)
            local repo_url="https://${REPO_DOMAIN}"

            sed -i "s|base_url:.*|base_url: \"${admin_url}\"|" "${CONFIG_DIR}/config.yaml"

            # Add repo_base_url if not present
            if grep -q "repo_base_url" "${CONFIG_DIR}/config.yaml"; then
                sed -i "s|repo_base_url:.*|repo_base_url: \"${repo_url}\"|" "${CONFIG_DIR}/config.yaml"
            else
                sed -i "/base_url:/a\\  repo_base_url: \"${repo_url}\"" "${CONFIG_DIR}/config.yaml"
            fi

            ok "Updated config:"
            ok "  base_url:      ${admin_url}"
            ok "  repo_base_url: ${repo_url}"
        fi

        # Firewall configuration
        configure_firewall "$ADMIN_PORT"

        # Start services
        if ask "Start rpmmanager and Caddy now?"; then
            systemctl enable --now rpmmanager
            ok "rpmmanager started"

            systemctl enable --now caddy
            ok "Caddy started"

            # Verify TLS certificates
            verify_caddy_tls "https://${ADMIN_DOMAIN}:${ADMIN_PORT}"

            echo ""
            info "Waiting for first-run password generation..."
            sleep 2
            local pw_line
            pw_line=$(journalctl -u rpmmanager --no-pager -n 50 2>/dev/null | grep "Generated admin password" || true)
            if [[ -n "$pw_line" ]]; then
                echo ""
                echo -e "  ${RED}${BOLD}${pw_line}${NC}"
                echo ""
            else
                warn "Could not find generated password. Check: journalctl -u rpmmanager | grep password"
            fi
        else
            echo ""
            info "Start manually when ready:"
            echo "  systemctl enable --now rpmmanager"
            echo "  systemctl enable --now caddy"
        fi
    else
        # No Caddy
        if [[ "$SKIP_SERVICE" == false ]] && command -v systemctl &>/dev/null; then
            echo ""
            if ask "Start rpmmanager now?"; then
                systemctl enable --now rpmmanager
                ok "rpmmanager started"

                sleep 2
                local pw_line
                pw_line=$(journalctl -u rpmmanager --no-pager -n 50 2>/dev/null | grep "Generated admin password" || true)
                if [[ -n "$pw_line" ]]; then
                    echo ""
                    echo -e "  ${RED}${BOLD}${pw_line}${NC}"
                    echo ""
                fi
            else
                echo ""
                info "Start manually:"
                echo "  systemctl enable --now rpmmanager"
            fi
        fi
    fi

    # ── Done ─────────────────────────────────────────────────
    echo ""
    echo -e "${GREEN}╔══════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║            Installation complete!                   ║${NC}"
    echo -e "${GREEN}╚══════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo "  Binary:   ${INSTALL_DIR}/rpmmanager"
    echo "  Config:   ${CONFIG_DIR}/config.yaml"
    echo "  Data:     ${DATA_DIR}/"

    if [[ "$setup_caddy" == true ]]; then
        echo "  Caddyfile: /etc/caddy/Caddyfile"
        echo ""
        echo -e "  ${BOLD}Public repo:${NC}   https://${REPO_DOMAIN}"
        echo -e "  ${BOLD}Admin panel:${NC}   https://${ADMIN_DOMAIN}:${ADMIN_PORT}"
        echo ""
        echo "  End users install packages via:"
        echo "    dnf config-manager --add-repo https://${REPO_DOMAIN}/<product>/<distro>/\$basearch/"
        echo ""
        echo "  Caddy auto-obtains Let's Encrypt certificates."
        echo "  Make sure DNS for both domains points to this server."
    else
        echo ""
        echo "  Admin panel: http://127.0.0.1:8080"
        echo ""
        echo "  To expose publicly, re-run with:"
        echo "    install.sh --domain rpms.example.com"
    fi

    echo ""
    echo "  First run generates a random admin password."
    echo "  View it:  journalctl -u rpmmanager | grep 'Generated admin password'"
    echo ""
}

main "$@"
