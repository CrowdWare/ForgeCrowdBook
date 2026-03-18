# Deployment Guide — ForgeCrowdBook

This guide describes how to deploy ForgeCrowdBook on a Linux VPS with a custom domain, automatic HTTPS via Caddy, and transactional email via Brevo.

---

## Prerequisites

- A Linux VPS (e.g. IONOS, Hetzner, Netcup) — 1 GB RAM is sufficient
- A domain or subdomain pointing to your VPS (e.g. `books.crowdware.info`)
- A free [Brevo](https://brevo.com) account for outgoing mail

---

## Step 1 — DNS

Point your subdomain to the VPS IP address. At your DNS provider, add an **A record**:

```
books.crowdware.info.  →  <your-vps-ip>
```

Wait for propagation (usually a few minutes, up to 24h).

---

## Step 2 — Brevo SMTP Setup

1. Create a free account at [brevo.com](https://brevo.com)
2. Go to **Settings → Senders & IP → Domains** and add your domain (`crowdware.info`)
3. Follow Brevo's instructions to add **SPF** and **DKIM** DNS records — this ensures your mails don't land in spam
4. Go to **SMTP & API → SMTP** and note your credentials:
   - Host: `smtp-relay.brevo.com`
   - Port: `587`
   - Login: your Brevo account email
   - Password: your Brevo SMTP API key

---

## Step 3 — Server Setup

SSH into your VPS and install the required tools.

### Install Go

```bash
wget https://go.dev/dl/go1.23.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.23.0.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
```

Check the [Go download page](https://go.dev/dl/) for the latest version.

### Install Caddy

```bash
sudo apt install -y debian-keyring debian-archive-keyring apt-transport-https curl
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | sudo gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | sudo tee /etc/apt/sources.list.d/caddy-stable.list
sudo apt update && sudo apt install caddy
```

---

## Step 4 — Deploy ForgeCrowdBook

### Clone and build

```bash
git clone https://codeberg.org/crowdware/forgecrowdbook.git
cd forgecrowdbook
go build -o forgecrowdbook .
```

### Create the config file

```bash
cp app-demo.sml app.sml
```

Edit `app.sml` — non-sensitive values only:

```sml
App {
    name: "ForgeCrowdBook"
    base_url: "https://books.crowdware.info"
    db: "./data/crowdbook.db"
    port: "8090"
    session_secret: ""
    admin_email: ""
    SMTP {
        host: "smtp-relay.brevo.com"
        port: "587"
        user: ""
        pass: ""
        from: "noreply@crowdware.info"
    }
}
```

### Generate a session secret

```bash
openssl rand -hex 32
# or: python3 -c "import secrets; print(secrets.token_hex(32))"
```

### Set environment variables

Create a file `/etc/forgecrowdbook.env` (readable by root only):

```bash
sudo nano /etc/forgecrowdbook.env
```

Contents:

```bash
FCB_SESSION_SECRET=<paste-generated-secret-here>
FCB_ADMIN_EMAIL=you@example.com
FCB_SMTP_USER=your-brevo-login@example.com
FCB_SMTP_PASS=your-brevo-smtp-api-key
```

Restrict permissions:

```bash
sudo chmod 600 /etc/forgecrowdbook.env
```

### Create the data directory

```bash
mkdir -p data
```

---

## Step 5 — Caddy (HTTPS)

Edit `/etc/caddy/Caddyfile`:

```
books.crowdware.info {
    reverse_proxy localhost:8090
}
```

Reload Caddy:

```bash
sudo systemctl reload caddy
```

Caddy automatically obtains and renews a Let's Encrypt TLS certificate. No manual HTTPS setup needed.

---

## Step 6 — Run as a systemd Service

Create `/etc/systemd/system/forgecrowdbook.service`:

```ini
[Unit]
Description=ForgeCrowdBook
After=network.target

[Service]
Type=simple
User=www-data
WorkingDirectory=/home/www-data/forgecrowdbook
EnvironmentFile=/etc/forgecrowdbook.env
ExecStart=/home/www-data/forgecrowdbook/forgecrowdbook
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable forgecrowdbook
sudo systemctl start forgecrowdbook
sudo systemctl status forgecrowdbook
```

---

## Step 7 — Verify

- Open `https://books.crowdware.info` in your browser
- Register with your admin email address
- Click the magic link in the mail
- You should land on the dashboard with admin access

---

## Updates

```bash
cd forgecrowdbook
git pull
go build -o forgecrowdbook .
sudo systemctl restart forgecrowdbook
```

---

## Troubleshooting

**App does not start:**
```bash
sudo journalctl -u forgecrowdbook -n 50
```

**Mail not arriving:**
- Check Brevo dashboard for send logs
- Verify SPF/DKIM DNS records are set correctly
- Check spam folder

**HTTPS not working:**
- Make sure DNS is pointing to the correct IP
- Check Caddy logs: `sudo journalctl -u caddy -n 50`
