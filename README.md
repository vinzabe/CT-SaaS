# Grant's Torrent

A modern, secure SaaS platform for converting torrent and magnet links to direct HTTP downloads.

Built as a complete rewrite of [jpillora/cloud-torrent](https://github.com/jpillora/cloud-torrent) with modern architecture, full SaaS features, and post-quantum cryptographic security.

## Quick Start

```bash
# Start Grant's Torrent (one command)
make up

# Stop (preserves data)
make down

# Stop and wipe all data
./stop.sh clean
```

**Access:**
- HTTP: http://localhost:7843
- HTTPS: https://localhost:7844
- API: http://localhost:7842

## Demo Accounts

| Account | Email | Password | Notes |
|---------|-------|----------|-------|
| **Admin** | `admin@grants.torrent` | `admin123` | Full access, admin panel |
| **Demo** | `demo@grants.torrent` | `demo123` | Restricted, 24hr retention, can't change settings |

The demo account is perfect for testing - downloads are automatically deleted after 24 hours and account settings are locked.

## Features

- **Torrent to Direct Link**: Convert any torrent or magnet link to a streamable HTTP download
- **Auto-ZIP**: Multi-file torrents are automatically zipped for easy download
- **SSL/HTTPS Support**: Built-in HTTPS support for secure connections
- **Post-Quantum Security**: Future-proof encryption using NIST-approved ML-DSA-65 algorithms
- **User Management**: Full authentication with JWT tokens and refresh token rotation
- **Subscription Plans**: Free, Starter, Pro, and Unlimited tiers with usage quotas
- **Admin Panel**: Manage users, view statistics, and monitor the platform
- **Real-time Updates**: Live progress updates via polling
- **Streaming Support**: Direct streaming with Range request support
- **Persistent Storage**: Database and downloads survive restarts

## Tech Stack

### Backend
- **Language**: Go 1.22
- **Framework**: Fiber v2
- **Torrent Engine**: anacrolix/torrent
- **Database**: PostgreSQL 16
- **Cache**: Redis 7
- **Auth**: JWT with Argon2id password hashing

### Frontend
- **Framework**: React 18 with TypeScript
- **Build Tool**: Vite
- **Styling**: Tailwind CSS
- **State Management**: Zustand + TanStack Query
- **UI Components**: Custom components with Lucide icons

## Ports

| Service | Port | Protocol | Description |
|---------|------|----------|-------------|
| Frontend (HTTP) | 7843 | HTTP | Web UI |
| Frontend (HTTPS) | 7844 | HTTPS | Web UI (SSL) |
| Backend API | 7842 | HTTP | REST API |
| BitTorrent | 42069 | TCP/UDP | Torrent protocol |

## SSL/HTTPS Support

Grant's Torrent includes built-in SSL support on port **7844**.

### Self-Signed Certificates (Default)
By default, self-signed certificates are generated automatically. These work for:
- Local development
- Cloudflare tunnels (with "No TLS Verify" option)
- Internal networks

### Custom Certificates
To use your own SSL certificates:

1. Create a `certs` directory:
```bash
mkdir -p certs
```

2. Add your certificates:
```bash
cp your-certificate.crt certs/server.crt
cp your-private-key.key certs/server.key
```

3. Mount the volume in `docker-compose.yml`:
```yaml
web:
  volumes:
    - ./certs:/etc/nginx/ssl:ro
```

4. Restart:
```bash
make down && make up
```

## Cloudflare Tunnel Setup

Grant's Torrent works great with Cloudflare Tunnels for secure public access.

### Option 1: HTTP Origin (Simplest)
Cloudflare handles SSL termination:

```bash
cloudflared tunnel --url http://localhost:7843
```

### Option 2: HTTPS Origin (End-to-End Encryption)
For full end-to-end encryption:

```bash
cloudflared tunnel --url https://localhost:7844 --no-tls-verify
```

Or with proper certificates (no `--no-tls-verify` needed):
```bash
cloudflared tunnel --url https://localhost:7844
```

### Persistent Tunnel Configuration

1. Create tunnel:
```bash
cloudflared tunnel create grants-torrent
```

2. Create config file (`~/.cloudflared/config.yml`):
```yaml
tunnel: grants-torrent
credentials-file: /root/.cloudflared/<tunnel-id>.json

ingress:
  - hostname: torrent.yourdomain.com
    service: https://localhost:7844
    originRequest:
      noTLSVerify: true
  - service: http_status:404
```

3. Route DNS:
```bash
cloudflared tunnel route dns grants-torrent torrent.yourdomain.com
```

4. Run tunnel:
```bash
cloudflared tunnel run grants-torrent
```

## Docker Containers

| Container | Image | Purpose |
|-----------|-------|---------|
| `grants-torrent-web` | nginx:alpine | Frontend server (HTTP + HTTPS) |
| `grants-torrent-api` | custom | Go backend |
| `grants-torrent-postgres` | postgres:16-alpine | Database |
| `grants-torrent-redis` | redis:7-alpine | Cache |

## Persistent Volumes

Data is stored in Docker volumes and survives restarts:

| Volume | Purpose |
|--------|---------|
| `torrent_postgres_data` | PostgreSQL database |
| `torrent_redis_data` | Redis cache |
| `torrent_downloads` | Downloaded torrent files |

## Development Setup

### Prerequisites
- Go 1.22+
- Node.js 20+
- Docker & Docker Compose

### Development Mode

```bash
# Start development databases
make dev-db

# Install dependencies
make install

# Run backend (terminal 1)
make dev-backend

# Run frontend (terminal 2)
make dev-frontend
```

Development URLs:
- Frontend: http://localhost:7843
- Backend API: http://localhost:7842

## API Endpoints

### Authentication
- `POST /api/v1/auth/register` - Create account
- `POST /api/v1/auth/login` - Login
- `POST /api/v1/auth/refresh` - Refresh token
- `POST /api/v1/auth/logout` - Logout
- `GET /api/v1/auth/me` - Get current user

### Torrents
- `POST /api/v1/torrents` - Add torrent (magnet or URL)
- `POST /api/v1/torrents/upload` - Upload .torrent file
- `GET /api/v1/torrents` - List torrents
- `GET /api/v1/torrents/:id` - Get torrent details
- `DELETE /api/v1/torrents/:id` - Delete torrent
- `POST /api/v1/torrents/:id/pause` - Pause download
- `POST /api/v1/torrents/:id/resume` - Resume download
- `POST /api/v1/torrents/:id/token` - Generate download token

### Downloads
- `GET /api/v1/download/:token` - Download file (public, token-authenticated)

### Admin
- `GET /api/v1/admin/users` - List users
- `GET /api/v1/admin/users/:id` - Get user details
- `PATCH /api/v1/admin/users/:id` - Update user
- `DELETE /api/v1/admin/users/:id` - Delete user
- `GET /api/v1/admin/torrents` - List all torrents
- `DELETE /api/v1/admin/torrents/:id` - Delete torrent
- `GET /api/v1/admin/stats` - Platform statistics
- `POST /api/v1/admin/cleanup` - Cleanup expired torrents

## Subscription Plans

| Plan | Price | Bandwidth | Concurrent | Retention |
|------|-------|-----------|------------|-----------|
| Free | $0/mo | 2 GB/mo | 1 | 24 hours |
| Starter | $5/mo | 50 GB/mo | 3 | 7 days |
| Pro | $15/mo | 500 GB/mo | 10 | 30 days |
| Unlimited | $30/mo | Unlimited | 25 | 90 days |

## Security Features

- **Argon2id**: OWASP-recommended password hashing
- **JWT with Rotation**: Short-lived access tokens (15 min) with refresh token rotation
- **Post-Quantum Cryptography**: ML-DSA-65 signatures for API security
- **TLS 1.2/1.3**: Modern SSL configuration with secure ciphers
- **HSTS**: HTTP Strict Transport Security enabled on HTTPS
- **Rate Limiting**: Per-user and per-IP rate limiting
- **Path Traversal Protection**: Secure file serving
- **Token-based Downloads**: Secure, expiring download links

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Backend server port | `7842` |
| `ENVIRONMENT` | `development` or `production` | `production` |
| `DATABASE_URL` | PostgreSQL connection string | Docker internal |
| `REDIS_URL` | Redis connection string | Docker internal |
| `JWT_SECRET` | JWT signing secret (auto-generated) | - |
| `JWT_ACCESS_EXPIRY` | Access token expiry (minutes) | `15` |
| `JWT_REFRESH_EXPIRY` | Refresh token expiry (days) | `7` |
| `DOWNLOAD_DIR` | Torrent download directory | `/downloads` |
| `TORRENT_PORT` | BitTorrent listen port | `42069` |

## Project Structure

```
grants-torrent/
├── backend/
│   ├── cmd/server/         # Application entry point
│   └── internal/
│       ├── auth/           # Authentication & JWT
│       ├── config/         # Configuration
│       ├── database/       # PostgreSQL layer
│       ├── handlers/       # HTTP handlers
│       ├── middleware/     # HTTP middleware
│       ├── models/         # Data models
│       └── torrent/        # Torrent engine & ZIP utility
├── frontend/
│   └── src/
│       ├── components/     # React components
│       ├── lib/            # API client & store
│       ├── pages/          # Page components
│       └── types/          # TypeScript types
├── docker/
│   ├── Dockerfile.backend
│   ├── Dockerfile.frontend
│   └── nginx.conf
├── docker-compose.yml
├── docker-compose.dev.yml
├── Makefile
├── start.sh
└── stop.sh
```

## Makefile Commands

| Command | Description |
|---------|-------------|
| `make up` | Start production (Docker) |
| `make down` | Stop all containers |
| `make dev` | Start development mode |
| `make dev-db` | Start dev databases only |
| `make dev-backend` | Run Go backend |
| `make dev-frontend` | Run React frontend |
| `make install` | Install all dependencies |
| `make build` | Build for production |
| `make test` | Run tests |
| `make docker-logs` | View container logs |

## Troubleshooting

### Containers won't start
```bash
# Full cleanup and restart
./stop.sh clean
make up
```

### Port already in use
Check if ports 7842, 7843, 7844, or 42069 are in use:
```bash
lsof -i :7842
lsof -i :7843
lsof -i :7844
```

### SSL Certificate Issues
If you get SSL errors with Cloudflare tunnel:
```bash
# Use --no-tls-verify for self-signed certs
cloudflared tunnel --url https://localhost:7844 --no-tls-verify
```

### View logs
```bash
docker logs grants-torrent-api
docker logs grants-torrent-web
```

### Database issues
```bash
# Connect to database
docker exec -it grants-torrent-postgres psql -U grantstorrent -d grantstorrent
```

## License

This project is licensed under the AGPL-3.0 License - see the LICENSE file for details.

## Acknowledgments

- [anacrolix/torrent](https://github.com/anacrolix/torrent) - Excellent Go torrent library
- [jpillora/cloud-torrent](https://github.com/jpillora/cloud-torrent) - Original inspiration
- [Fiber](https://gofiber.io/) - Fast Go web framework
- [Tailwind CSS](https://tailwindcss.com/) - Utility-first CSS framework
- [Cloudflare](https://cloudflare.com/) - Tunnel and security services
