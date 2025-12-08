#!/bin/bash
set -euo pipefail

echo "=== KumoMTA + KumoMTA-UI Installer (Rocky Linux 9) ==="

if [ "$EUID" -ne 0 ]; then
  echo "Please run as root (sudo -i)."
  exit 1
fi

PANEL_DIR="/opt/kumomta-ui"
BIN_NAME="kumomta-ui-server"
BIN_PATH="$PANEL_DIR/$BIN_NAME"
DB_DIR="/var/lib/kumomta-ui"
SERVICE_FILE="/etc/systemd/system/kumomta-ui.service"
NGINX_CONF="/etc/nginx/conf.d/kumomta-ui.conf"

echo
read -rp "Set system hostname (eg mta.yourdomain.com) [leave empty to skip]: " SYS_HOSTNAME
read -rp "Panel domain for HTTPS (eg mta.yourdomain.com) [leave empty for HTTP on :9000]: " PANEL_DOMAIN

LE_EMAIL=""
if [ -n "$PANEL_DOMAIN" ]; then
  read -rp "Email for Let's Encrypt (eg admin@yourdomain.com): " LE_EMAIL
  if [ -z "$LE_EMAIL" ]; then
    echo "Let's Encrypt email is required when PANEL_DOMAIN is set."
    exit 1
  fi
fi

echo
echo "[*] Verifying panel directory at $PANEL_DIR ..."
if [ ! -d "$PANEL_DIR" ]; then
  echo "Directory $PANEL_DIR not found."
  echo "Clone your Git repo there, e.g.:"
  echo "  sudo mkdir -p /opt/kumomta-ui"
  echo "  sudo git clone https://github.com/pulak-ranjan/kumomta-ui.git /opt/kumomta-ui"
  exit 1
fi

cd "$PANEL_DIR"

# --------------------------
# System hostname
# --------------------------
if [ -n "$SYS_HOSTNAME" ]; then
  echo "[*] Setting system hostname to $SYS_HOSTNAME"
  hostnamectl set-hostname "$SYS_HOSTNAME" || echo "Warning: failed to set hostname"
fi

# --------------------------
# Base dependencies
# --------------------------
echo "[*] Installing base dependencies (git, Go, firewalld, epel-release, dnf-plugins-core, SELinux tools)..."
dnf install -y git golang firewalld epel-release dnf-plugins-core policycoreutils-python-utils curl

# Make sure firewalld is running
systemctl enable --now firewalld || true

# Disable postfix if present (can conflict with KumoMTA)
echo "[*] Disabling postfix if present..."
systemctl disable --now postfix 2>/dev/null || true

# --------------------------
# Install Dovecot & Fail2ban
# --------------------------
echo "[*] Installing Dovecot and Fail2ban..."
dnf install -y dovecot fail2ban fail2ban-firewalld || true

echo "[*] Enabling Fail2ban (recommended for security)..."
systemctl enable --now fail2ban 2>/dev/null || true

# (Dovecot is installed but not auto-started; enable if you need IMAP/POP)
# systemctl enable --now dovecot

# --------------------------
# Install Node.js (for frontend)
# --------------------------
if ! command -v node >/dev/null 2>&1; then
  echo "[*] Installing Node.js 20..."
  dnf module install -y nodejs:20 || dnf install -y nodejs npm
else
  echo "[*] Node.js already installed."
fi

# --------------------------
# Install nginx + certbot if domain is provided
# --------------------------
if [ -n "$PANEL_DOMAIN" ]; then
  echo "[*] Installing nginx and certbot (from EPEL)..."
  dnf install -y nginx certbot python3-certbot-nginx
fi

# --------------------------
# Install KumoMTA
# --------------------------
echo "[*] Adding KumoMTA repository and installing KumoMTA..."
dnf config-manager --add-repo https://openrepo.kumomta.com/files/kumomta-rocky.repo || true
yum install -y kumomta

echo "[*] Ensuring KumoMTA directories exist..."
mkdir -p /opt/kumomta/etc/policy
mkdir -p /opt/kumomta/etc/dkim

# --------------------------
# Build Go backend
# --------------------------
echo "[*] Running go mod tidy..."
GO111MODULE=on go mod tidy

echo "[*] Building KumoMTA-UI backend binary..."
GO111MODULE=on go build -o "$BIN_PATH" ./cmd/server
chmod +x "$BIN_PATH"

# --------------------------
# Build frontend (React/Vite)
# --------------------------
echo "[*] Building frontend..."
if [ -d "$PANEL_DIR/web" ]; then
  cd "$PANEL_DIR/web"
  npm install
  npm run build
  cd "$PANEL_DIR"
else
  echo "Warning: web directory not found, skipping frontend build"
fi

# --------------------------
# DB directory
# --------------------------
echo "[*] Creating DB directory at $DB_DIR ..."
mkdir -p "$DB_DIR"
chmod 755 "$DB_DIR"

# --------------------------
# systemd service for kumomta-ui
# --------------------------
echo "[*] Creating systemd service at $SERVICE_FILE ..."
cat >"$SERVICE_FILE" <<EOF
[Unit]
Description=KumoMTA UI Backend
After=network.target

[Service]
User=root
Group=root
WorkingDirectory=$PANEL_DIR
Environment=DB_DIR=$DB_DIR
ExecStart=$BIN_PATH
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

# --------------------------
# SELinux: allow executing binary from /opt/kumomta-ui
# --------------------------
echo "[*] Applying SELinux context for $BIN_PATH ..."
if command -v semanage >/dev/null 2>&1; then
  semanage fcontext -a -t bin_t "${PANEL_DIR}(/.*)?" 2>/dev/null || semanage fcontext -m -t bin_t "${PANEL_DIR}(/.*)?"
  restorecon -Rv "$PANEL_DIR" || true
fi
chcon -t bin_t "$BIN_PATH" || true

# --------------------------
# SELinux: allow nginx to talk to backend on 9000
# --------------------------
if command -v semanage >/dev/null 2>&1; then
  echo "[*] Allowing httpd_t (nginx) to connect to network + port 9000 via SELinux..."
  setsebool -P httpd_can_network_connect on || true
  semanage port -a -t http_port_t -p tcp 9000 2>/dev/null || semanage port -m -t http_port_t -p tcp 9000 || true
fi

# --------------------------
# systemd: reload & enable services
# --------------------------
echo "[*] Reloading systemd daemon..."
systemctl daemon-reload

echo "[*] Enabling and starting KumoMTA daemon..."
systemctl enable --now kumomta || true

echo "[*] Enabling KumoMTA-UI service..."
systemctl enable --now kumomta-ui || true

# --------------------------
# nginx configuration (if DOMAIN)
# --------------------------
if [ -n "$PANEL_DOMAIN" ]; then
  echo "[*] Writing nginx config to $NGINX_CONF ..."
  cat >"$NGINX_CONF" <<EOF
server {
    listen 80;
    server_name $PANEL_DOMAIN;

    root $PANEL_DIR/web/dist;
    index index.html;

    location / {
        try_files \$uri /index.html;
    }

    location /api/ {
        proxy_pass http://127.0.0.1:9000/api/;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }
}
EOF

  echo "[*] Testing nginx config..."
  nginx -t

  echo "[*] Enabling and starting nginx..."
  systemctl enable --now nginx

  echo "[*] Configuring firewalld for HTTP/HTTPS..."
  firewall-cmd --permanent --add-service=http || true
  firewall-cmd --permanent --add-service=https || true
  firewall-cmd --reload || true

  echo "[*] Requesting Let's Encrypt certificate via certbot..."
  certbot --nginx -d "$PANEL_DOMAIN" --non-interactive --agree-tos -m "$LE_EMAIL" --redirect || echo "Warning: certbot failed; check DNS and try manually."

else
  echo "[*] No PANEL_DOMAIN provided: running UI directly on http://SERVER_IP:9000"
  echo "[*] Opening port 9000 in firewalld..."
  firewall-cmd --permanent --add-port=9000/tcp || true
  firewall-cmd --reload || true
fi

# --------------------------
# Final info
# --------------------------
VPS_IP=""
if command -v ip >/dev/null 2>&1; then
  VPS_IP=$(ip route get 1.1.1.1 2>/dev/null | awk '/src/ {for(i=1;i<=NF;i++) if ($i=="src") print $(i+1)}' | head -n1)
fi
if [ -z "$VPS_IP" ] && command -v hostname >/dev/null 2>&1; then
  VPS_IP=$(hostname -I 2>/dev/null | awk '{print $1}')
fi

echo
echo "==========================================="
echo "  KumoMTA + KumoMTA-UI Installed"
echo "==========================================="
echo

if [ -n "$PANEL_DOMAIN" ]; then
  echo "Panel URL:  https://$PANEL_DOMAIN/"
  echo "API URL:    https://$PANEL_DOMAIN/api"
  if [ -n "$VPS_IP" ]; then
    echo
    echo "DNS reminder: point A record $PANEL_DOMAIN -> $VPS_IP"
  fi
else
  if [ -n "$VPS_IP" ]; then
    echo "Panel URL:  http://$VPS_IP:9000/"
    echo "API URL:    http://$VPS_IP:9000/api"
  else
    echo "Panel URL:  http://<YOUR_SERVER_IP>:9000/"
    echo "API URL:    http://<YOUR_SERVER_IP>:9000/api"
  fi
fi

echo
echo "Next steps:"
echo "  1) Open the Panel URL in your browser"
echo "  2) Use 'First-time Setup' to create the admin user"
echo
echo "Useful commands:"
echo "  systemctl status kumomta-ui"
echo "  journalctl -u kumomta-ui -f"
echo "  systemctl restart kumomta-ui"
echo "  systemctl status kumomta"
echo
