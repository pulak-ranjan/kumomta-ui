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

# 1. Basic checks
if [ ! -d "$PANEL_DIR" ]; then
  echo "Directory $PANEL_DIR not found."
  echo "Clone your Git repo there, e.g.:"
  echo "  git clone https://github.com/pulak-ranjan/kumomta-ui.git $PANEL_DIR"
  exit 1
fi

cd "$PANEL_DIR"

# 2. Install dependencies
echo "[*] Installing dependencies (git, Go, firewalld)..."
dnf install -y git golang firewalld

# 3. Build Go binary
echo "[*] Building Go binary..."
GO111MODULE=on go build -o "$BIN_PATH" ./cmd/server

chmod +x "$BIN_PATH"

# 4. Ensure DB directory exists
echo "[*] Creating DB directory: $DB_DIR"
mkdir -p "$DB_DIR"
chmod 755 "$DB_DIR"

# 5. Ensure Kumo directories exist (if Kumo is already installed, this will be no-op)
echo "[*] Ensuring Kumo policy and DKIM directories exist..."
mkdir -p /opt/kumomta/etc/policy
mkdir -p /opt/kumomta/etc/dkim

# 6. Create systemd service
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

# 7. Open firewall port 9000 (optional: you can later restrict or proxy via nginx)
echo "[*] Configuring firewall to allow port 9000/tcp..."
systemctl enable firewalld --now || true
firewall-cmd --permanent --add-port=9000/tcp || true
firewall-cmd --reload || true

# 8. Start service
echo "[*] Enabling and starting kumomta-ui.service..."
systemctl daemon-reload
systemctl enable kumomta-ui
systemctl restart kumomta-ui

sleep 2
systemctl status kumomta-ui --no-pager || true

echo "=== Installation complete ==="
echo "Backend should be accessible at: http://<VPS_IP>:9000"
echo "Use /api/auth/register to create the first admin."
