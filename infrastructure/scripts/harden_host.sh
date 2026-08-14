#!/bin/bash
set -e

# ==============================================================================
# NAS.AI Host Hardening Script
# Usage: sudo ./harden_host.sh
# ==============================================================================

if [ "$EUID" -ne 0 ]; then
  echo "Please run as root"
  exit 1
fi

echo "🚀 Starting System Hardening..."

# ------------------------------------------------------------------------------
# 1. SSH Hardening
# ------------------------------------------------------------------------------
echo "🔒 Securing SSH..."

cp /etc/ssh/sshd_config /etc/ssh/sshd_config.bak.$(date +%F_%T)

# Apply settings using sed to ensures policies are set correctly
sed -i 's/^#*PermitRootLogin.*/PermitRootLogin no/' /etc/ssh/sshd_config
sed -i 's/^#*PasswordAuthentication.*/PasswordAuthentication no/' /etc/ssh/sshd_config
sed -i 's/^#*ChallengeResponseAuthentication.*/ChallengeResponseAuthentication no/' /etc/ssh/sshd_config
sed -i 's/^#*UsePAM.*/UsePAM yes/' /etc/ssh/sshd_config
sed -i 's/^#*PubkeyAuthentication.*/PubkeyAuthentication yes/' /etc/ssh/sshd_config

# Validate config before restart
if sshd -t; then
    systemctl restart sshd
    echo "✅ SSH Hardened & Restarted"
else
    echo "❌ SSH Configuration invalid! Reverting..."
    mv /etc/ssh/sshd_config.bak.$(date +%F_%T) /etc/ssh/sshd_config
    systemctl restart sshd
    exit 1
fi

# ------------------------------------------------------------------------------
# 2. Firewall (UFW) Configuration
# ------------------------------------------------------------------------------
echo "🛡️ Configuring Firewall (UFW)..."

# Reset strict defaults
ufw --force reset
ufw default deny incoming
ufw default allow outgoing

# Allow Loopback (Localhost)
ufw allow in on lo

# Allow Critical Services
ufw allow 22/tcp comment 'SSH Access'
ufw allow 80/tcp comment 'HTTP WebUI'
ufw allow 443/tcp comment 'HTTPS WebUI'

# DNS Config: Outgoing is ALLOWED by default.
# Only uncomment below if acting as a DNS Server (e.g. AdGuard/PiHole)
# ufw allow 53/udp comment 'DNS Service'
# ufw allow 53/tcp comment 'DNS Service'

# Enable Firewall
ufw --force enable
echo "✅ UFW Firewall Enabled. Incoming traffic default blocked."

# ------------------------------------------------------------------------------
# 3. Fail2Ban Setup
# ------------------------------------------------------------------------------
echo "🚫 Installing & Configuring Fail2Ban..."

# Install if missing (Debian/Ubuntu)
if ! command -v fail2ban-client &> /dev/null; then
    apt-get update && apt-get install -y fail2ban
fi

# Configure Jail
cat > /etc/fail2ban/jail.local <<EOL
[DEFAULT]
bantime  = 1h
findtime = 10m
maxretry = 5
ignoreip = 127.0.0.1/8

[sshd]
enabled = true
port    = ssh
filter  = sshd
logpath = /var/log/auth.log
maxretry = 3
EOL

systemctl enable fail2ban
systemctl restart fail2ban
echo "✅ Fail2Ban installed and active for SSH."

echo "🎉 Security Hardening Complete!"
