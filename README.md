# XMen Uploader

A modern, fast, and private file uploader built with Go and vanilla web technologies. Features an Apple Liquid Glass UI, drag-and-drop file sharing, custom expiration times, download limits, and an administrative dashboard.

---

## Features

- **No Account Required**: Instant drag-and-drop file sharing with automatic share links.
- **Configurable Limits**: Set maximum file upload size and maximum retention duration.
- **Download Limits**: Restrict how many times a file can be downloaded before it is deleted.
- **Automatic Expiration**: Files self-destruct when their deadline passes. Background workers clean up expired files from disk and database.
- **Admin Dashboard**: Live overview of active files, total downloads, and storage usage.
- **Hardened Security**:
  - Header-based admin authentication (`Authorization: Bearer <secret>`).
  - Rate limiting on API routes and brute-force protection.
  - Proxy-aware IP spoofing protection (`TRUST_PROXY`).
  - Atomic download tracking to prevent race conditions.
  - Path traversal protections and strict security headers (`CSP`, `nosniff`, `DENY`).
- **Apple Liquid Glass UI**: Refined translucent glass materials, responsive layout, and automatic light/dark mode.

---

## Installation (Ubuntu / Debian)

Install the latest release with one command on any systemd-based Linux host:

```bash
curl -fsSL https://raw.githubusercontent.com/4xmen/uploader/main/install.sh | sudo bash -s -- --install
```

### Upgrade Existing Installation

Upgrade the binary to the latest release without affecting your data or configuration:

```bash
curl -fsSL https://raw.githubusercontent.com/4xmen/uploader/main/install.sh | sudo bash -s -- --update
```

### Service Management

```bash
# Check service status
sudo systemctl status xmen-uploader

# View real-time logs
sudo journalctl -u xmen-uploader -f

# Restart service after editing .env
sudo systemctl restart xmen-uploader
```

---

## Configuration

Configuration is stored in `/opt/xmen-uploader/.env`:

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8089` | Port for web interface and API |
| `MAX_UPLOAD_SIZE_MB` | `50` | Maximum file size in megabytes |
| `MAX_ALLOWED_LIFETIME_DAYS` | `7` | Maximum file retention allowed in days |
| `ADMIN_SECRET` | *(auto-generated)* | Secret key used to access `/admin` |
| `DATABASE_PATH` | `/opt/xmen-uploader/data/uploader.db` | Path to SQLite database file |
| `STORAGE_PATH` | `/opt/xmen-uploader/data/uploads` | Path to stored uploaded files |
| `ALLOWED_ORIGINS` | `*` | Allowed CORS origins for API requests |
| `TRUST_PROXY` | `false` | Set to true only if running behind a trusted reverse proxy (e.g. Nginx, Caddy) |

---

## Admin Dashboard

Access the admin dashboard at:
```
http://<server-ip>:8089/admin
```
Enter the `ADMIN_SECRET` from `/opt/xmen-uploader/.env` to unlock live statistics.
