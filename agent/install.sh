#!/bin/sh
# ============================================================
# Remote WoL — OpenWrt Agent Kurulum Betiği
# Bu betik OpenWrt modem üzerinde çalıştırılmalıdır.
# ============================================================

set -e

echo "╔══════════════════════════════════════════╗"
echo "║   Remote WoL — OpenWrt Agent Kurulumu    ║"
echo "╚══════════════════════════════════════════╝"
echo ""

# ─── Yapılandırma ───
VPS_IP="${1:-SUNUCU_IP_ADRESI}"
RATHOLE_TOKEN="${2:-change-me-agent-secret}"
AGENT_TOKEN="${3:-change-me-agent-secret}"
AGENT_PORT="${4:-8080}"
RATHOLE_PORT="${5:-2333}"

# ─── 1. Gerekli paketleri yükle ───
echo "📦 Paketler kontrol ediliyor..."
opkg update
opkg install etherwake curl uhttpd 2>/dev/null || true

# ─── 2. CGI dosyalarını oluştur ───
echo "📁 CGI dosyaları kuruluyor..."
mkdir -p /www/wol-agent/cgi-bin

# Wake CGI
cat > /www/wol-agent/cgi-bin/wake << 'EOF'
#!/bin/sh
echo "Content-Type: application/json"
echo ""

eval $(echo "$QUERY_STRING" | tr '&' '\n' | while read pair; do
    key=$(echo "$pair" | cut -d= -f1)
    val=$(echo "$pair" | cut -d= -f2-)
    echo "$key=$val"
done)

[ -f /etc/wol-agent.conf ] && . /etc/wol-agent.conf
AGENT_TOKEN="${AGENT_TOKEN:-change-me-agent-secret}"
if [ "$token" != "$AGENT_TOKEN" ]; then
    echo '{"success":false,"message":"unauthorized"}'
    exit 0
fi

if [ -z "$mac" ]; then
    echo '{"success":false,"message":"mac parameter required"}'
    exit 0
fi

if [ -z "$iface" ]; then
    iface="br-lan"
fi

echo "$mac" | grep -qE '^([0-9A-Fa-f]{2}:){5}[0-9A-Fa-f]{2}$'
if [ $? -ne 0 ]; then
    echo '{"success":false,"message":"invalid MAC address format"}'
    exit 0
fi

if etherwake -i "$iface" "$mac" 2>/dev/null; then
    echo "{\"success\":true,\"message\":\"Magic packet sent to $mac via $iface\"}"
else
    if ether-wake -i "$iface" "$mac" 2>/dev/null; then
        echo "{\"success\":true,\"message\":\"Magic packet sent to $mac via $iface\"}"
    else
        echo '{"success":false,"message":"etherwake command failed"}'
    fi
fi
EOF
chmod +x /www/wol-agent/cgi-bin/wake

# Ping CGI
cat > /www/wol-agent/cgi-bin/ping << 'EOF'
#!/bin/sh
echo "Content-Type: application/json"
echo ""

eval $(echo "$QUERY_STRING" | tr '&' '\n' | while read pair; do
    key=$(echo "$pair" | cut -d= -f1)
    val=$(echo "$pair" | cut -d= -f2-)
    echo "$key=$val"
done)

[ -f /etc/wol-agent.conf ] && . /etc/wol-agent.conf
AGENT_TOKEN="${AGENT_TOKEN:-change-me-agent-secret}"
if [ "$token" != "$AGENT_TOKEN" ]; then
    echo '{"alive":false,"ip":"","error":"unauthorized"}'
    exit 0
fi

if [ -z "$ip" ]; then
    echo '{"alive":false,"ip":"","error":"ip parameter required"}'
    exit 0
fi

echo "$ip" | grep -qE '^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}$'
if [ $? -ne 0 ]; then
    echo '{"alive":false,"ip":"","error":"invalid IP format"}'
    exit 0
fi

if ping -c 1 -W 1 "$ip" > /dev/null 2>&1; then
    echo "{\"alive\":true,\"ip\":\"$ip\"}"
else
    echo "{\"alive\":false,\"ip\":\"$ip\"}"
fi
EOF
chmod +x /www/wol-agent/cgi-bin/ping

# Health CGI
cat > /www/wol-agent/cgi-bin/health << 'EOF'
#!/bin/sh
echo "Content-Type: application/json"
echo ""

eval $(echo "$QUERY_STRING" | tr '&' '\n' | while read pair; do
    key=$(echo "$pair" | cut -d= -f1)
    val=$(echo "$pair" | cut -d= -f2-)
    echo "$key=$val"
done)

[ -f /etc/wol-agent.conf ] && . /etc/wol-agent.conf
AGENT_TOKEN="${AGENT_TOKEN:-change-me-agent-secret}"
if [ "$token" != "$AGENT_TOKEN" ]; then
    echo '{"status":"unauthorized"}'
    exit 0
fi

HOSTNAME=$(cat /proc/sys/kernel/hostname 2>/dev/null || echo "unknown")
UPTIME=$(cat /proc/uptime 2>/dev/null | awk '{printf "%d saat %d dakika", $1/3600, ($1%3600)/60}')
LOAD=$(cat /proc/loadavg 2>/dev/null | awk '{print $1}')

echo "{\"status\":\"ok\",\"hostname\":\"$HOSTNAME\",\"uptime\":\"$UPTIME\",\"load\":\"$LOAD\"}"
EOF
chmod +x /www/wol-agent/cgi-bin/health

# ─── 3. Agent token'ını ayarla ───
echo "🔑 Agent token yapılandırılıyor..."
cat > /etc/wol-agent.conf <<CONF
export AGENT_TOKEN="$AGENT_TOKEN"
CONF
chmod 600 /etc/wol-agent.conf

# ─── 4. HTTP Agent servisi oluştur ───
echo "🌐 HTTP Agent servisi oluşturuluyor..."
cat > /etc/init.d/wol-agent <<'SERVICE'
#!/bin/sh /etc/rc.common
START=99
STOP=10

start() {
    . /etc/wol-agent.conf
    export AGENT_TOKEN

    # Try uhttpd first (default on OpenWrt), fallback to busybox httpd
    if command -v uhttpd >/dev/null 2>&1; then
        uhttpd -p 127.0.0.1:8080 -h /www/wol-agent -c /cgi-bin &
        echo "WoL Agent started via uhttpd on port 8080"
    elif command -v httpd >/dev/null 2>&1; then
        httpd -p 127.0.0.1:8080 -h /www/wol-agent
        echo "WoL Agent started via busybox httpd on port 8080"
    else
        echo "ERROR: Neither uhttpd nor httpd found!"
        exit 1
    fi
}

stop() {
    # Kill only the instance on 8080 so we don't break LuCI web interface
    PID=$(netstat -tlpn 2>/dev/null | grep ':8080 ' | awk '{print $7}' | cut -d'/' -f1)
    if [ -n "$PID" ]; then
        kill "$PID" 2>/dev/null
    fi
    echo "WoL Agent stopped"
}
SERVICE

chmod +x /etc/init.d/wol-agent

# ─── 5. Rathole client kurulumu ───
echo "🔌 Rathole indiriliyor..."

# Cihaz mimarisini algıla
ARCH=$(uname -m)
case "$ARCH" in
    mips*)   RATHOLE_ARCH="mips-unknown-linux-musl" ;;
    aarch64) RATHOLE_ARCH="aarch64-unknown-linux-musl" ;;
    armv7*)  RATHOLE_ARCH="armv7-unknown-linux-musleabihf" ;;
    x86_64)  RATHOLE_ARCH="x86_64-unknown-linux-musl" ;;
    *)       echo "⚠️ Desteklenmeyen mimari: $ARCH"; exit 1 ;;
esac

RATHOLE_VERSION="v0.5.0"
RATHOLE_URL="https://github.com/rapiz1/rathole/releases/download/${RATHOLE_VERSION}/rathole-${RATHOLE_ARCH}.zip"

curl -L -o rathole.zip "$RATHOLE_URL"
unzip -o rathole.zip
mv rathole /usr/bin/rathole
chmod +x /usr/bin/rathole
rm -f rathole.zip

# ─── 6. Rathole client yapılandırması ───
echo "📝 Rathole yapılandırılıyor..."
mkdir -p /etc/rathole

cat > /etc/rathole/client.toml <<TOML
[client]
remote_addr = "${VPS_IP}:${RATHOLE_PORT}"

[client.services.wol-agent]
token = "${RATHOLE_TOKEN}"
local_addr = "127.0.0.1:${AGENT_PORT}"
TOML

# ─── 7. Rathole init.d servisi ───
cat > /etc/init.d/rathole <<'SERVICE'
#!/bin/sh /etc/rc.common
START=98
STOP=11

start() {
    killall rathole 2>/dev/null || true
    sleep 1
    /usr/bin/rathole --client /etc/rathole/client.toml >/dev/null 2>&1 &
    echo "Rathole client started"
}

stop() {
    killall rathole 2>/dev/null
    echo "Rathole client stopped"
}
SERVICE

chmod +x /etc/init.d/rathole

# ─── 8. Servisleri etkinleştir ve başlat ───
echo "🚀 Servisler başlatılıyor..."
/etc/init.d/wol-agent enable
/etc/init.d/wol-agent restart
/etc/init.d/rathole enable
/etc/init.d/rathole restart

echo ""
echo "╔══════════════════════════════════════════╗"
echo "║      ✅ Kurulum tamamlandı!              ║"
echo "║                                          ║"
echo "║  Agent:   127.0.0.1:${AGENT_PORT}              ║"
echo "║  Rathole: ${VPS_IP}:${RATHOLE_PORT}      ║"
echo "╚══════════════════════════════════════════╝"
