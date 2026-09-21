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
echo "📦 Paketler yükleniyor..."
opkg update
opkg install etherwake curl

# ─── 2. CGI dosyalarını kopyala ───
echo "📁 CGI dosyaları kuruluyor..."
mkdir -p /www/wol-agent/cgi-bin

# Wake script
cp www/cgi-bin/wake /www/wol-agent/cgi-bin/wake
chmod +x /www/wol-agent/cgi-bin/wake

# Ping script
cp www/cgi-bin/ping /www/wol-agent/cgi-bin/ping
chmod +x /www/wol-agent/cgi-bin/ping

# Health script
cp www/cgi-bin/health /www/wol-agent/cgi-bin/health
chmod +x /www/wol-agent/cgi-bin/health

# ─── 3. Agent token'ını ayarla ───
echo "🔑 Agent token yapılandırılıyor..."
cat > /etc/wol-agent.conf <<CONF
export AGENT_TOKEN="$AGENT_TOKEN"
CONF
chmod 600 /etc/wol-agent.conf

# ─── 4. BusyBox httpd servisi oluştur ───
echo "🌐 HTTP Agent servisi oluşturuluyor..."
cat > /etc/init.d/wol-agent <<'SERVICE'
#!/bin/sh /etc/rc.common
START=99
STOP=10

start() {
    . /etc/wol-agent.conf
    export AGENT_TOKEN
    httpd -p 127.0.0.1:8080 -h /www/wol-agent
    echo "WoL Agent started on port 8080"
}

stop() {
    killall httpd 2>/dev/null
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
    /usr/bin/rathole --client /etc/rathole/client.toml &
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
/etc/init.d/wol-agent start
/etc/init.d/rathole enable
/etc/init.d/rathole start

echo ""
echo "╔══════════════════════════════════════════╗"
echo "║      ✅ Kurulum tamamlandı!              ║"
echo "║                                          ║"
echo "║  Agent:   127.0.0.1:${AGENT_PORT}              ║"
echo "║  Rathole: ${VPS_IP}:${RATHOLE_PORT}      ║"
echo "╚══════════════════════════════════════════╝"
