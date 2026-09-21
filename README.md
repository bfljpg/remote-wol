# 🌐 Remote WoL — Uzaktan Wake-on-LAN Yönetim Paneli

Evinizdeki bilgisayarları internet üzerinden tek tıkla uyandırın. OpenWrt modem + VPS + Rathole tüneli ile güvenli uzaktan WoL.

## 🏗️ Mimari

```
┌─────────────────────┐         ┌─────────────────────────────────┐
│   👤 Kullanıcı       │         │  ☁️  VPS (Uzak Sunucu)           │
│   (Tarayıcı)        │ ──────▶ │  ┌───────────┐ ┌─────────────┐ │
│                     │  HTTPS  │  │ Go Web App │ │ Rathole Srv │ │
└─────────────────────┘         │  │ :3000      │ │ :2333       │ │
                                │  └─────┬─────┘ └──────┬──────┘ │
                                │        │ localhost:9090│        │
                                └────────┼───────────────┼────────┘
                                         │               │
                                    ─────┼───────────────┼──── İnternet
                                         │               │
                                ┌────────┼───────────────┼────────┐
                                │  🏠 Ev │Ağı            │        │
                                │  ┌─────┴─────┐ ┌──────┴──────┐ │
                                │  │ WoL Agent  │ │Rathole Clnt │ │
                                │  │ :8080 (CGI)│ │             │ │
                                │  └─────┬─────┘ └─────────────┘ │
                                │        │ etherwake              │
                                │  ┌─────┴─────┐                 │
                                │  │ 🖥️ PC      │                 │
                                │  │ (Magic Pkt)│                 │
                                │  └───────────┘                 │
                                └─────────────────────────────────┘
```

## ✨ Özellikler

- 🔐 JWT tabanlı kimlik doğrulama
- 📱 Birden fazla cihaz yönetimi (ekle, düzenle, sil)
- ⚡ Tek tıkla Wake-on-LAN
- 🔄 Gerçek zamanlı cihaz durum takibi (açık/kapalı)
- 📡 Rathole ile güvenli NAT traversal
- 🎨 Modern dark glassmorphism arayüz
- 🐳 Docker ile kolay deployment
- 📦 Tek binary — `go:embed` ile frontend gömülü

## 🚀 Hızlı Başlangıç

### 1. VPS Kurulumu

#### Docker ile (Önerilen)

```bash
# Repo'yu klonla
git clone https://github.com/kullanici/remote-wol.git
cd remote-wol

# .env dosyasını oluştur
cp server/.env.example .env
# .env dosyasını düzenle — JWT_SECRET, INITIAL_ADMIN_PASS, RATHOLE_TOKEN, AGENT_TOKEN değerlerini belirleyin!
# (Tüm servisler —Go backend ve Rathole tüneli— ayarlarını tek bir .env dosyasından alır)

# Başlat
docker compose up -d
```

#### Manuel (Docker'sız)

```bash
cd server
cp .env.example .env
# .env dosyasını düzenle

go build -ldflags="-s -w" -o remote-wol .
./remote-wol
```

### 2. OpenWrt Modem Kurulumu

```bash
# Agent dosyalarını modeme kopyala
scp -r agent/ root@MODEM_IP:/tmp/wol-setup/

# Modeme bağlan ve kurulumu çalıştır
ssh root@MODEM_IP
cd /tmp/wol-setup
chmod +x install.sh
./install.sh VPS_IP_ADRESI RATHOLE_TOKEN AGENT_TOKEN
```

**Parametreler:**
| # | Parametre | Açıklama | Varsayılan |
|---|-----------|----------|------------|
| 1 | `VPS_IP` | VPS sunucunuzun IP adresi | — |
| 2 | `RATHOLE_TOKEN` | Rathole tünel şifresi | change-me-agent-secret |
| 3 | `AGENT_TOKEN` | Agent API şifresi | change-me-agent-secret |
| 4 | `AGENT_PORT` | Agent HTTP portu | 8080 |
| 5 | `RATHOLE_PORT` | Rathole bağlantı portu | 2333 |

### 3. Kullanım

1. Tarayıcınızda `http://VPS_IP:3000` adresine gidin
2. Varsayılan giriş: `admin` / `admin` (⚠️ hemen değiştirin!)
3. "Cihaz Ekle" ile bilgisayarınızın MAC ve IP bilgilerini girin
4. "Uyandır" butonuna tıklayın 🚀

## 📁 Proje Yapısı

```
remote-wol/
├── server/                      # Go backend + embedded frontend
│   ├── main.go                  # Giriş noktası
│   ├── go.mod
│   ├── internal/
│   │   ├── config/config.go     # Yapılandırma
│   │   ├── db/db.go             # SQLite veritabanı
│   │   ├── auth/auth.go         # JWT auth
│   │   ├── handlers/devices.go  # Device API handlers
│   │   ├── agent/agent.go       # OpenWrt agent istemcisi
│   │   └── middleware/middleware.go
│   └── public/                  # Frontend (go:embed)
│       ├── index.html
│       ├── css/style.css
│       └── js/app.js
├── agent/                       # OpenWrt tarafı
│   ├── www/cgi-bin/
│   │   ├── wake                 # WoL tetikleme
│   │   ├── ping                 # Durum kontrolü
│   │   └── health               # Sağlık kontrolü
│   └── install.sh               # Otomatik kurulum
├── rathole/                     # Tünel yapılandırması
│   ├── server.toml
│   ├── client.toml
│   └── rathole-server.service
├── Dockerfile
├── docker-compose.yml
└── README.md
```

## 🔧 API Endpoints

| Method | Path | Açıklama |
|--------|------|----------|
| POST | `/api/auth/login` | Giriş yap ve JWT al |
| POST | `/api/auth/change-password` | Şifre değiştir (JWT gerekli) |
| GET | `/api/devices` | Cihaz listesi |
| POST | `/api/devices` | Yeni cihaz ekle |
| PUT | `/api/devices/{id}` | Cihaz güncelle |
| DELETE | `/api/devices/{id}` | Cihaz sil |
| POST | `/api/devices/{id}/wake` | WoL paketi gönder |
| GET | `/api/devices/{id}/status` | Cihaz durumu sorgula |
| GET | `/api/devices/status/all` | Toplu durum sorgusu |
| GET | `/api/devices/agent/health` | Agent bağlantı durumu |

## ⚠️ Güvenlik

- `.env` dosyasındaki `JWT_SECRET` değerini mutlaka değiştirin
- `INITIAL_ADMIN_PASS` şifresini güçlü yapın (yalnızca ilk kurulumda veritabanını tohumlamak için kullanılır)
- Admin kimlik bilgileri SQLite veritabanında bcrypt hash olarak saklanır; dilediğiniz zaman API üzerinden şifrenizi güncelleyebilirsiniz
- `AGENT_TOKEN` değerlerinin VPS ve modemde aynı olduğundan emin olun
- Rathole `server.toml` ve `client.toml` dosyalarındaki token'ları eşleştirin
- VPS üzerinde firewall kurallarıyla sadece gerekli portları açın (3000, 2333)

## 📄 Lisans

MIT
