#!/usr/bin/env bash
# Partir Core Deployment Script
# Usage: ./scripts/deploy.sh [options]
#
# Options:
#   --install     Install binaries and systemd units
#   --upgrade     Upgrade binaries only
#   --migrate     Run database migrations
#   --start       Start services
#   --stop        Stop services
#   --status      Check service status
#   --full        Full deployment (install + migrate + start)

set -euo pipefail

# Configuration
PARTIR_USER="${PARTIR_USER:-partir}"
PARTIR_GROUP="${PARTIR_GROUP:-partir}"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
CONFIG_DIR="${CONFIG_DIR:-/etc/partir}"
DATA_DIR="${DATA_DIR:-/var/lib/partir}"
LOG_DIR="${LOG_DIR:-/var/log/partir}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root"
        exit 1
    fi
}

create_user() {
    if ! id "$PARTIR_USER" &>/dev/null; then
        log_info "Creating user $PARTIR_USER..."
        useradd --system --no-create-home --shell /sbin/nologin "$PARTIR_USER"
    fi
}

create_directories() {
    log_info "Creating directories..."
    mkdir -p "$CONFIG_DIR"
    mkdir -p "$DATA_DIR"
    mkdir -p "$LOG_DIR"
    chown -R "$PARTIR_USER:$PARTIR_GROUP" "$DATA_DIR" "$LOG_DIR"
}

install_binaries() {
    log_info "Building binaries..."
    
    # Build all binaries
    go build -ldflags="-w -s" -o /tmp/partir-build/alpha ./cmd/alpha
    go build -ldflags="-w -s" -o /tmp/partir-build/beta ./cmd/beta
    go build -ldflags="-w -s" -o /tmp/partir-build/foundry ./cmd/foundry
    go build -ldflags="-w -s" -o /tmp/partir-build/omega ./cmd/omega
    go build -ldflags="-w -s" -o /tmp/partir-build/partir ./cmd/partir
    go build -ldflags="-w -s" -o /tmp/partir-build/guard ./cmd/guard
    go build -ldflags="-w -s" -o /tmp/partir-build/migrate ./cmd/migrate
    
    log_info "Installing binaries to $INSTALL_DIR..."
    install -m 755 /tmp/partir-build/alpha "$INSTALL_DIR/"
    install -m 755 /tmp/partir-build/beta "$INSTALL_DIR/"
    install -m 755 /tmp/partir-build/foundry "$INSTALL_DIR/"
    install -m 755 /tmp/partir-build/omega "$INSTALL_DIR/"
    install -m 755 /tmp/partir-build/partir "$INSTALL_DIR/"
    install -m 755 /tmp/partir-build/guard "$INSTALL_DIR/"
    install -m 755 /tmp/partir-build/migrate "$INSTALL_DIR/"
    
    rm -rf /tmp/partir-build
    log_info "Binaries installed successfully"
}

install_systemd_units() {
    log_info "Installing systemd units..."
    
    cp deploy/systemd/partir-health.service /etc/systemd/system/
    cp deploy/systemd/partir-foundry.service /etc/systemd/system/
    
    systemctl daemon-reload
    log_info "Systemd units installed"
}

install_config() {
    if [[ ! -f "$CONFIG_DIR/partir.env" ]]; then
        log_info "Installing default environment config..."
        cp .env.example "$CONFIG_DIR/partir.env"
        chmod 600 "$CONFIG_DIR/partir.env"
        chown "$PARTIR_USER:$PARTIR_GROUP" "$CONFIG_DIR/partir.env"
        log_warn "Please edit $CONFIG_DIR/partir.env with your production values!"
    else
        log_info "Config file already exists, skipping..."
    fi
}

run_migrations() {
    log_info "Running database migrations..."
    
    # Source environment
    if [[ -f "$CONFIG_DIR/partir.env" ]]; then
        set -a
        source "$CONFIG_DIR/partir.env"
        set +a
    fi
    
    "$INSTALL_DIR/migrate" up
    log_info "Migrations complete"
}

start_services() {
    log_info "Starting services..."
    systemctl enable partir-health
    systemctl start partir-health
    log_info "Services started"
}

stop_services() {
    log_info "Stopping services..."
    systemctl stop partir-health 2>/dev/null || true
    systemctl stop partir-foundry 2>/dev/null || true
    log_info "Services stopped"
}

show_status() {
    echo ""
    log_info "Service Status:"
    systemctl status partir-health --no-pager 2>/dev/null || log_warn "partir-health not installed"
    echo ""
    
    log_info "Binary Versions:"
    "$INSTALL_DIR/partir" version 2>/dev/null || log_warn "partir not installed"
    echo ""
    
    log_info "Health Check:"
    "$INSTALL_DIR/partir" healthz 2>/dev/null || log_warn "Health check failed"
}

do_install() {
    check_root
    create_user
    create_directories
    install_binaries
    install_systemd_units
    install_config
    log_info "Installation complete!"
}

do_upgrade() {
    check_root
    stop_services
    install_binaries
    start_services
    log_info "Upgrade complete!"
}

do_full() {
    do_install
    run_migrations
    start_services
    show_status
    log_info "Full deployment complete!"
}

# Parse arguments
case "${1:-}" in
    --install)
        do_install
        ;;
    --upgrade)
        do_upgrade
        ;;
    --migrate)
        run_migrations
        ;;
    --start)
        start_services
        ;;
    --stop)
        stop_services
        ;;
    --status)
        show_status
        ;;
    --full)
        do_full
        ;;
    *)
        echo "Usage: $0 [--install|--upgrade|--migrate|--start|--stop|--status|--full]"
        echo ""
        echo "Options:"
        echo "  --install     Install binaries and systemd units"
        echo "  --upgrade     Upgrade binaries only"
        echo "  --migrate     Run database migrations"
        echo "  --start       Start services"
        echo "  --stop        Stop services"
        echo "  --status      Check service status"
        echo "  --full        Full deployment (install + migrate + start)"
        exit 1
        ;;
esac
