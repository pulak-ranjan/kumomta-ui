#!/bin/bash
set -e

echo "=== Kumo UI Backend Installer (Rocky Linux 9) ==="

if [ "$EUID" -ne 0 ]; then
  echo "Please run as root."
  exit 1
fi

PANEL_DIR="/opt/kumomta-ui"
BIN_NAME="kumomta-ui-server"
BIN_PATH="$PANEL_DIR/$BIN_NAME"
DB_DIR="/var/lib/kumomta-ui"
SERVICE_FILE="/etc/systemd/system/kumomta-ui.service"

# --- Ask basic questions ---

echo
read -rp "Set system hostname (eg mta.yourdomain.com) [leave empty to skip]: " SYS_HOSTNAME
read -rp "Panel domain for HTTPS (eg mta.yourdomain.com) [leave empty for no HTTPS]: " PANEL_DOMAIN

LE_EMAIL=""
if [ -n "$PANEL_DOMAIN" ]; then
  read -rp "Email for Let's Encrypt (eg admin@yourdomain.com): " LE_EMAIL
  if [ -z "$LE_EMAIL" ]; then
    echo "Let's Encrypt email is required when PANEL_DOMAIN is set."
    exit 1
  fi
fi

echo

# --- Basic checks ---

if [ ! -d "$PANEL_DIR" ]; then
  echo "Directory $PANEL_DIR not found."
  echo "Clone your Git repo there, e.g.:"
  echo "  git clone https://github.com/pulak-ranjan/kumomta-ui.git $PANEL_DIR"
  exit 1
fi

cd "$PANEL_DIR"

# --- Set system hostname if provided ---

if [ -n "$SYS_HOSTNAME" ]; then
  echo "[*] Setting system hostname to $SYS_HOSTNAME"
  hostnamectl set-hostname "$SYS_HOSTNAME" || echo "Warning: failed to set hostname"
fi

# --- Install dependencies ---

echo "[*] Installing dependencies (git, Go, firewalld)..."
dnf install -y git golang firewalld

if [ -n "$PANEL_DOMAIN" ]; then
  echo "[*] Installing nginx and certbot..."
  dnf install -y nginx certbot python3-certbot-nginx
fi

# --- Build Go binary ---

echo "[*] Building Go binary..."
GO111MODULE=on go build -o "$BIN_PATH" ./cmd/server
chmod +x "$BIN_PATH"

# --- Ensure DB directory exists ---

echo "[*] Creating DB directory: $DB_DIR"
mkdir -p "$DB_DIR"
chmod 755 "$DB_DIR"

# --- Ensure Kumo directories exist (if already installed, this is no-op) ---

echo "[*] Ensuring Kumo policy and DKIM directories exist..."
mkdir -p /opt/kumomta/etc/policy
mkdir -p /opt/kumomta/etc/dkim

# --- Create systemd service ---

echo "[*] Creating systemd service at $SERVICE_FILE"

cat >"$SERVICE_FILE" <<EOF
[Unit]
Description=Kumo UI Backend
After=network.target

[Service]
User=root
Group=root
WorkingDirectory=$PANEL_DIR
ExecStart=$BIN_PATH
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

# --- Firewall configuration ---

echo "[*] Configuring firewall..."

systemctl enable firewalld --now || true

if [ -n "$PANEL_DOMAIN" ]; then
  # Using nginx + HTTPS; open 80 and 443
  firewall-cmd --permanent --add-service=http || true
  firewall-cmd --permanent --add-service=https || true
  # No need to expose 9000 in this mode
else
  # No domain/HTTPS: expose :9000 directly
  firewall-cmd --permanent --add-port=9000/tcp || true
fi

firewall-cmd --reload || true

# --- Start backend service ---

echo "[*] Enabling and starting kumomta-ui.service..."
systemctl daemon-reload
systemctl enable kumomta-ui
systemctl restart kumomta-ui

sleep 2
systemctl status kumomta-ui --no-pager || true

# --- Configure nginx + HTTPS if domain provided ---

if [ -n "$PANEL_DOMAIN" ]; then
  echo "[*] Configuring nginx reverse proxy for $PANEL_DOMAIN -> 127.0.0.1:9000"

  NGINX_CONF="/etc/nginx/conf.d/kumomta-ui.conf"

  cat >"$NGINX_CONF" <<EOF
server {
    listen 80;
    server_name $PANEL_DOMAIN;

    location / {
        proxy_pass http://127.0.0.1:9000;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }
}
EOF

  systemctl enable nginx --now
  nginx -t && systemctl reload nginx

  echo "[*] Requesting Let's Encrypt certificate for $PANEL_DOMAIN..."
  echo "    Make sure DNS A record points to this server BEFORE this step."

  certbot --nginx \
    -d "$PANEL_DOMAIN" \
    -m "$LE_EMAIL" \
    --agree-tos \
    --non-interactive \
    --redirect || echo "Warning: certbot failed; you can rerun certbot manually later."
fi

# --- Detect primary server IP ---

VPS_IP=""
if command -v ip >/dev/null 2>&1; then
  VPS_IP=$(ip route get 1.1.1.1 2>/dev/null | awk '/src/ {for(i=1;i<=NF;i++) if ($i=="src") print $(i+1)}' | head -n1)
fi

if [ -z "$VPS_IP" ] && command -v hostname >/dev/null 2>&1; then
  VPS_IP=$(hostname -I 2>/dev/null | awk '{print $1}')
fi

# --- Final message ---

echo "=== Installation complete ==="

if [ -n "$PANEL_DOMAIN" ]; then
  echo "Panel URL:  https://$PANEL_DOMAIN/"
  echo "API URL:    https://$PANEL_DOMAIN/api"
  echo
  echo "DNS reminder: ensure an A record points $PANEL_DOMAIN -> $VPS_IP"
else
  if [ -n "$VPS_IP" ]; then
    echo "Panel URL:  http://$VPS_IP:9000/"
    echo "API URL:    http://$VPS_IP:9000/api"
  else
    echo "Could not auto-detect VPS IP."
    echo "Panel URL (example):  http://<your-vps-ip>:9000/"
    echo "API URL (example):    http://<your-vps-ip>:9000/api"
  fi
fi

echo
echo "Open the Panel URL in your browser and use 'First-time Setup' to create the admin user."
