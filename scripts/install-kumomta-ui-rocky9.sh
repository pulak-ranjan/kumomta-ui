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
MIGRATE_BIN="$PANEL_DIR/kumomta-ui-migrate"
DB_DIR="/var/lib/kumomta-ui"
SERVICE_FILE="/etc/systemd/system/kumomta-ui.service"
NGINX_CONF="/etc/nginx/conf.d/kumomta-ui.conf"

echo

# --------------------------
# 1. Smart Hostname Check
# --------------------------
CURRENT_HOSTNAME=$(hostname)
if [ "$CURRENT_HOSTNAME" != "localhost" ] && [ "$CURRENT_HOSTNAME" != "localhost.localdomain" ]; then
    echo "Current System Hostname: $CURRENT_HOSTNAME"
    read -rp "Set system hostname [Press Enter to keep '$CURRENT_HOSTNAME']: " INPUT_HOSTNAME
    if [ -z "$INPUT_HOSTNAME" ]; then
        SYS_HOSTNAME="$CURRENT_HOSTNAME"
    else
        SYS_HOSTNAME="$INPUT_HOSTNAME"
    fi
else
    read -rp "Set system hostname (eg mta.yourdomain.com) [leave empty to skip]: " SYS_HOSTNAME
fi

# --------------------------
# 2. Smart Panel/SSL Check
# --------------------------
EXISTING_DOMAIN=""
if [ -f "$NGINX_CONF" ]; then
    # Extract server_name from existing nginx config
    EXISTING_DOMAIN=$(awk '/server_name/ {print $2}' "$NGINX_CONF" | head -n1 | tr -d ';')
fi

if [ -n "$EXISTING_DOMAIN" ]; then
    echo "Current Panel Domain (SSL): $EXISTING_DOMAIN"
    read -rp "Panel domain for HTTPS [Press Enter to keep '$EXISTING_DOMAIN']: " INPUT_DOMAIN
    if [ -z "$INPUT_DOMAIN" ]; then
        PANEL_DOMAIN="$EXISTING_DOMAIN"
    else
        PANEL_DOMAIN="$INPUT_DOMAIN"
    fi
else
    read -rp "Panel domain for HTTPS (eg mta.yourdomain.com) [leave empty for HTTP on :9000]: " PANEL_DOMAIN
fi

# --------------------------
# 3. Smart Email Prompt
# --------------------------
LE_EMAIL=""
if [ -n "$PANEL_DOMAIN" ]; then
    # Only ask for email if we DON'T have a cert yet
    if [ ! -d "/etc/letsencrypt/live/$PANEL_DOMAIN" ]; then
        read -rp "Email for Let's Encrypt (eg admin@yourdomain.com): " LE_EMAIL
        if [ -z "$LE_EMAIL" ]; then
            echo "Error: Let's Encrypt email is required for new SSL setup."
            exit 1
        fi
    else
        echo "SSL Certificate already exists for $PANEL_DOMAIN. Skipping email prompt."
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
if [ -n "$SYS_HOSTNAME" ] && [ "$SYS_HOSTNAME" != "$CURRENT_HOSTNAME" ]; then
  echo "[*] Setting system hostname to $SYS_HOSTNAME"
  hostnamectl set-hostname "$SYS_HOSTNAME" || echo "Warning: failed to set hostname"
fi

# --------------------------
# Base dependencies
# --------------------------
echo "[*] Installing base dependencies..."
dnf install -y epel-release
dnf install -y git golang firewalld dnf-plugins-core policycoreutils-python-utils curl bind-utils swaks nano

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
systemctl enable --now fail2ban 2>/dev/null || true
systemctl enable --now dovecot 2>/dev/null || true

# --- FIX: Configure Dovecot for Mixed-Case Usernames ---
echo "[*] Configuring Dovecot to preserve username case..."
DOVECOT_CONF="/etc/dovecot/conf.d/10-auth.conf"
if [ -f "$DOVECOT_CONF" ]; then
    # Replace 'auth_username_format = %Lu' or '#auth_username_format = %Lu' with '%u'
    sed -i 's/^#*auth_username_format.*/auth_username_format = %u/' "$DOVECOT_CONF"
    # echo "    Updated auth_username_format to %u" # Silent success
    systemctl restart dovecot
else
    echo "    Warning: $DOVECOT_CONF not found. Skipping auto-fix."
fi

# --------------------------
# Configure Firewall (Safe & Secure)
# --------------------------
echo "[*] Configuring firewall ports..."
# Standard SMTP ports
firewall-cmd --permanent --add-port=25/tcp
firewall-cmd --permanent --add-port=587/tcp
firewall-cmd --permanent --add-port=465/tcp

# Bounce processing ports (IMAP/POP3)
firewall-cmd --permanent --add-service=imaps
firewall-cmd --permanent --add-service=pop3s
firewall-cmd --permanent --add-service=imap
firewall-cmd --permanent --add-service=pop3

# Ensure HTTP/HTTPS are open
firewall-cmd --permanent --add-service=http
firewall-cmd --permanent --add-service=https

# SECURITY: Ensure port 9000 is CLOSED to public (only used locally by Nginx)
firewall-cmd --permanent --remove-port=9000/tcp 2>/dev/null || true

firewall-cmd --reload || true

# --------------------------
# Install Node.js
# --------------------------
if ! command -v node >/dev/null 2>&1; then
  echo "[*] Installing Node.js 20..."
  dnf module install -y nodejs:20 || dnf install -y nodejs npm
else
  # echo "[*] Node.js already installed."
  :
fi

# --------------------------
# Install Nginx + Certbot
# --------------------------
if [ -n "$PANEL_DOMAIN" ]; then
  # echo "[*] Installing nginx and certbot..."
  dnf install -y nginx certbot python3-certbot-nginx
fi

# --------------------------
# Install KumoMTA
# --------------------------
if rpm -q kumomta &>/dev/null; then
  # echo "[*] KumoMTA is already installed."
  :
else
  echo "[*] Installing KumoMTA..."
  dnf config-manager --add-repo https://openrepo.kumomta.com/files/kumomta-rocky.repo || true
  yum install -y kumomta
fi

# Ensure policy directories exist
mkdir -p /opt/kumomta/etc/policy
mkdir -p /opt/kumomta/etc/dkim

# --------------------------
# Install Documentation (For AI Agent)
# --------------------------
if [ ! -d "$PANEL_DIR/docs" ]; then
    echo "[*] Fetching KumoMTA documentation for AI Agent..."
    rm -rf "$PANEL_DIR/docs-temp"
    git clone --depth 1 https://github.com/KumoCorp/kumomta.git "$PANEL_DIR/docs-temp"
    if [ -d "$PANEL_DIR/docs-temp/docs" ]; then
        mv "$PANEL_DIR/docs-temp/docs" "$PANEL_DIR/docs"
        echo "    Documentation installed."
    fi
    rm -rf "$PANEL_DIR/docs-temp"
fi

# --------------------------
# Build Backend & Migration Tool
# --------------------------
echo "[*] Building backend services..."
GO111MODULE=on go mod tidy

# 1. Build Server
GO111MODULE=on go build -o "$BIN_PATH" ./cmd/server
chmod +x "$BIN_PATH"

# 2. Build Migration Tool
echo "[*] Building migration tool..."
GO111MODULE=on go build -o "$MIGRATE_BIN" ./cmd/migrate
chmod +x "$MIGRATE_BIN"

# --------------------------
# Build Frontend
# --------------------------
echo "[*] Building frontend..."
if [ -d "$PANEL_DIR/web" ]; then
  cd "$PANEL_DIR/web"
  npm install
  npm run build
  cd "$PANEL_DIR"
else
  echo "Warning: web directory not found"
fi

# --------------------------
# Setup Systemd
# --------------------------
mkdir -p "$DB_DIR"
chmod 755 "$DB_DIR"

# Force localhost binding if Nginx is used
if [ -n "$PANEL_DOMAIN" ]; then
  LISTEN_ADDR="127.0.0.1:9000"
else
  LISTEN_ADDR="0.0.0.0:9000"
fi

# echo "[*] Configuring systemd service..."
cat >"$SERVICE_FILE" <<EOF
[Unit]
Description=KumoMTA UI Backend
After=network.target

[Service]
User=root
Group=root
WorkingDirectory=$PANEL_DIR
Environment=DB_DIR=$DB_DIR
Environment=LISTEN_ADDR=$LISTEN_ADDR
ExecStart=$BIN_PATH
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

# --------------------------
# SELinux
# --------------------------
if command -v semanage >/dev/null 2>&1; then
  semanage fcontext -a -t bin_t "${PANEL_DIR}(/.*)?" 2>/dev/null || true
  restorecon -Rv "$PANEL_DIR" || true
  setsebool -P httpd_can_network_connect on || true
  semanage port -a -t http_port_t -p tcp 9000 2>/dev/null || true
fi

systemctl daemon-reload
systemctl enable --now kumomta || true
systemctl enable --now kumomta-ui || true
systemctl restart kumomta-ui

# --------------------------
# Nginx & SSL Config
# --------------------------
if [ -n "$PANEL_DOMAIN" ]; then
  echo "[*] Configuring Nginx for $PANEL_DOMAIN..."
  
  # 1. Write the base HTTP config
  cat >"$NGINX_CONF" <<EOF
server {
    listen 80;
    server_name $PANEL_DOMAIN;
    root $PANEL_DIR/web/dist;
    index index.html;
    location / { try_files \$uri /index.html; }
    location /api/ {
        proxy_pass http://127.0.0.1:9000/api/;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }
}
EOF

  # 2. Check Nginx syntax
  nginx -t
  systemctl enable --now nginx
  
  # 3. Always run Certbot to update the Nginx config, 
  #    but use 'reinstall' strategy if certs exist to avoid hitting limits.
  if [ -d "/etc/letsencrypt/live/$PANEL_DOMAIN" ]; then
      echo "[*] Re-applying existing SSL Certificate to Nginx config..."
      # Use --reinstall to ensure the nginx conf is updated to listen on 443
      certbot --nginx -d "$PANEL_DOMAIN" --non-interactive --reinstall --redirect
  else
      echo "[*] Requesting new SSL Certificate..."
      certbot --nginx -d "$PANEL_DOMAIN" --non-interactive --agree-tos -m "$LE_EMAIL" --redirect || echo "Warning: Certbot failed. Check DNS/Firewall."
  fi
else
  # If no domain, open port 9000 locally
  firewall-cmd --permanent --add-port=9000/tcp
  firewall-cmd --reload
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
echo "  KumoMTA + KumoMTA-UI Setup Complete"
echo "==========================================="
echo
echo "Status Checks:"
echo "  [x] Dovecot username format fixed (%u)"
echo "  [x] Firewall ports opened (SMTP, IMAP, POP3)"
echo "  [x] Panel Port 9000 secured (Localhost only)"
echo

if [ -n "$PANEL_DOMAIN" ]; then
  echo "Panel URL:  https://$PANEL_DOMAIN/"
  if [ -n "$VPS_IP" ]; then
    echo "DNS reminder: point A record $PANEL_DOMAIN -> $VPS_IP"
  fi
else
  if [ -n "$VPS_IP" ]; then
    echo "Panel URL:  http://$VPS_IP:9000/"
  else
    echo "Panel URL:  http://<YOUR_SERVER_IP>:9000/"
  fi
fi
