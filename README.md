# CT-SaaS

A modern, secure SaaS platform for converting torrent and magnet links to direct HTTP downloads.

Built as a complete rewrite of [jpillora/cloud-torrent](https://github.com/jpillora/cloud-torrent) with modern architecture, full SaaS features, and post-quantum cryptographic security.

## Quick Start

```bash
# Start CT-SaaS (one command)
make up

# Stop (preserves data)
make down

# Stop and wipe all data
./stop.sh clean
```

**Access (HTTPS enabled by default):**
- Frontend: https://localhost:7843 or https://localhost:7844
- API: http://localhost:7842

> Note: Self-signed SSL certificates are generated automatically. Your browser will show a security warning - this is normal for self-signed certs.

## Demo Accounts

| Account | Email | Password | Notes |
|---------|-------|----------|-------|
| **Admin** | `admin@ct.saas` | `admin123` | Full access, admin panel |
| **Demo** | `demo@ct.saas` | `demo123` | Restricted, 24hr retention, can't change settings |

The demo account is perfect for testing - downloads are automatically deleted after 24 hours and account settings are locked.

## Features

- **Torrent to Direct Link**: Convert any torrent or magnet link to a streamable HTTP download
- **Auto-ZIP**: Multi-file torrents are automatically zipped for easy download
- **SSL/HTTPS Always On**: Both frontend ports use HTTPS with TLS 1.2/1.3
- **Post-Quantum Security**: Future-proof encryption using NIST-approved ML-DSA-65 algorithms
- **User Management**: Full authentication with JWT tokens and refresh token rotation
- **Subscription Plans**: Free, Starter, Pro, and Unlimited tiers with usage quotas
- **Admin Panel**: Manage users, view statistics, and monitor the platform
- **Real-time Updates**: Live progress updates via Server-Sent Events (SSE)
- **Streaming Support**: Direct streaming with Range request support
- **Persistent Storage**: Database and downloads survive restarts

## Recent Changes

### v1.1.0 - SSE Real-Time Updates
- **Server-Sent Events (SSE)**: Replaced polling with real-time push-based updates
  - Live torrent progress without page refresh
  - Connection status indicator (Live/Offline) in dashboard
  - Auto-reconnect on connection drop
  - Reduced server load (polling reduced from 3s to 30s fallback)
- **SSE Endpoints**:
  - `GET /api/v1/events` - User torrent updates (requires auth)
  - `GET /api/v1/admin/events` - All torrent updates (admin only)
- **Token-based SSE Auth**: Supports query parameter authentication for browser EventSource compatibility

### v1.0.0 - Initial Release
- Full torrent-to-HTTP conversion
- User authentication with JWT
- Admin panel
- SSL/HTTPS support
- Cloudflare Tunnel compatible
- Demo account system

## Security

### Overview

CT-SaaS implements multiple layers of security to protect user data and ensure secure operations.

### Authentication & Authorization

| Feature | Implementation | Details |
|---------|----------------|---------|
| Password Hashing | Argon2id | OWASP-recommended, memory-hard algorithm |
| Access Tokens | JWT (HS256) | 15-minute expiry, signed with secure secret |
| Refresh Tokens | Secure random | 7-day expiry, SHA-256 hashed storage |
| Token Rotation | Automatic | New refresh token on each use |
| Rate Limiting | Per-user/IP | 100 requests per minute |

### Transport Security

| Feature | Implementation | Details |
|---------|----------------|---------|
| TLS Version | 1.2 / 1.3 | Modern protocols only |
| HTTPS | Always-on | Both frontend ports encrypted |
| HSTS | Enabled | Forces HTTPS connections |
| Secure Ciphers | Modern suite | AES-GCM, ChaCha20-Poly1305 |
| Certificate | Auto-generated | Self-signed for dev, supports custom certs |

### Cryptographic Security

| Feature | Algorithm | Standard |
|---------|-----------|----------|
| Password Hash | Argon2id | OWASP recommended |
| JWT Signing | HMAC-SHA256 | RFC 7519 |
| Post-Quantum | ML-DSA-65 | NIST FIPS 204 |
| Download Tokens | Secure Random | 256-bit entropy |
| Refresh Tokens | SHA-256 | Hashed storage |

### Post-Quantum Cryptography

CT-SaaS includes optional post-quantum cryptographic support using NIST-approved algorithms:

- **ML-DSA-65** (Module-Lattice Digital Signature Algorithm): Used for API request signing
- **Security Level**: NIST Level 3 (equivalent to AES-192)
- **Implementation**: Cloudflare's CIRCL library

This provides protection against future quantum computer attacks while maintaining compatibility with current systems.

### Application Security

| Protection | Description |
|------------|-------------|
| Path Traversal | All file paths validated against download directory |
| SQL Injection | Parameterized queries via pgx |
| XSS | React's built-in escaping + CSP headers |
| CSRF | Token-based authentication |
| Token Expiry | Download links expire in 24 hours |
| Download Limits | Max 10 downloads per token |
| File Validation | .torrent extension required for uploads |

### Secure Headers

All responses include security headers:
```
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
Referrer-Policy: strict-origin-when-cross-origin
Strict-Transport-Security: max-age=31536000; includeSubDomains
```

### Security Best Practices

1. **Change Default Credentials**: Update demo/admin passwords in production
2. **Set JWT_SECRET**: Use a secure, random 64+ character secret
3. **Use Real Certificates**: Replace self-signed certs with Let's Encrypt or similar
4. **Firewall Rules**: Only expose ports 7843/7844, keep 7842 internal
5. **Regular Updates**: Keep dependencies updated for security patches

### Reporting Security Issues

If you discover a security vulnerability, please report it responsibly by opening a private issue or contacting the maintainers directly. Do not disclose security issues publicly until a fix is available.

## Tech Stack

### Backend
- **Language**: Go 1.22
- **Framework**: Fiber v2
- **Torrent Engine**: anacrolix/torrent
- **Database**: PostgreSQL 16
- **Cache**: Redis 7
- **Auth**: JWT with Argon2id password hashing
- **Real-time**: Server-Sent Events (SSE)

### Frontend
- **Framework**: React 18 with TypeScript
- **Build Tool**: Vite
- **Styling**: Tailwind CSS
- **State Management**: Zustand + TanStack Query
- **UI Components**: Custom components with Lucide icons
- **Real-time**: EventSource API for SSE

## Ports

| Service | Port | Protocol | Description |
|---------|------|----------|-------------|
| Frontend | 7843 | **HTTPS** | Web UI (SSL) |
| Frontend | 7844 | **HTTPS** | Web UI (SSL) - Alternative |
| Backend API | 7842 | HTTP | REST API (internal) |
| BitTorrent | 42069 | TCP/UDP | Torrent protocol |

**Both frontend ports (7843 and 7844) use HTTPS/SSL by default.**

## SSL/HTTPS Configuration

CT-SaaS has SSL enabled by default on both frontend ports.

### Self-Signed Certificates (Default)
Self-signed certificates are generated automatically during container build. These work for:
- Local development
- Cloudflare tunnels (with "No TLS Verify" option)
- Internal networks

### Custom Certificates
To use your own SSL certificates (e.g., from Let's Encrypt):

1. Create a `certs` directory:
```bash
mkdir -p certs
```

2. Add your certificates:
```bash
cp your-certificate.crt certs/server.crt
cp your-private-key.key certs/server.key
```

3. Add volume mount to `docker-compose.yml`:
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

CT-SaaS is designed to work with Cloudflare Tunnels for secure public access.

### Recommended: HTTPS Origin
For end-to-end encryption with Cloudflare tunnel:

```bash
# Quick tunnel (temporary)
cloudflared tunnel --url https://localhost:7844 --no-tls-verify

# Or use port 7843
cloudflared tunnel --url https://localhost:7843 --no-tls-verify
```

The `--no-tls-verify` flag is needed for self-signed certificates.

### Persistent Tunnel Configuration

1. Create tunnel:
```bash
cloudflared tunnel create ct-saas
```

2. Create config file (`~/.cloudflared/config.yml`):
```yaml
tunnel: ct-saas
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
cloudflared tunnel route dns ct-saas torrent.yourdomain.com
```

4. Run tunnel:
```bash
cloudflared tunnel run ct-saas
```

### With Custom Certificates
If using real certificates (not self-signed), remove `noTLSVerify`:

```yaml
ingress:
  - hostname: torrent.yourdomain.com
    service: https://localhost:7844
  - service: http_status:404
```

## Docker Containers

| Container | Image | Purpose |
|-----------|-------|---------|
| `ct-saas-web` | nginx:alpine | Frontend server (HTTPS) |
| `ct-saas-api` | custom | Go backend |
| `ct-saas-postgres` | postgres:16-alpine | Database |
| `ct-saas-redis` | redis:7-alpine | Cache |

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
- Frontend: http://localhost:5173 (Vite dev server)
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

### Real-time Events (SSE)
- `GET /api/v1/events?token=<jwt>` - Subscribe to torrent updates
- `GET /api/v1/admin/events?token=<jwt>` - Subscribe to all updates (admin)

SSE events:
- `connected` - Connection established
- `torrents` - Torrent status updates (progress, speed, peers)
- `heartbeat` - Keep-alive signal (every second)
- `timeout` - Connection timeout (reconnect required)

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
ct-saas/
├── backend/
│   ├── cmd/server/         # Application entry point
│   └── internal/
│       ├── auth/           # Authentication & JWT & PQC
│       ├── config/         # Configuration
│       ├── database/       # PostgreSQL layer
│       ├── handlers/       # HTTP handlers (incl. SSE)
│       ├── middleware/     # HTTP middleware
│       ├── models/         # Data models
│       └── torrent/        # Torrent engine & ZIP utility
├── frontend/
│   └── src/
│       ├── components/     # React components
│       ├── hooks/          # Custom hooks (incl. useSSE)
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

### Browser shows "Not Secure" warning
This is normal with self-signed certificates. Click "Advanced" and "Proceed" to continue. For production, use real certificates.

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

### Cloudflare tunnel 502 error
Make sure to use `--no-tls-verify` with self-signed certificates:
```bash
cloudflared tunnel --url https://localhost:7844 --no-tls-verify
```

### SSE not connecting
- Check if you're authenticated (valid JWT token)
- Check browser console for connection errors
- Verify the API is accessible at `/api/v1/events`
- Check for proxy/firewall blocking long-lived connections

### View logs
```bash
docker logs ct-saas-api
docker logs ct-saas-web
```

### Database issues
```bash
# Connect to database
docker exec -it ct-saas-postgres psql -U grantstorrent -d grantstorrent
```

## License

This project is licensed under the AGPL-3.0 License - see the LICENSE file for details.

## Acknowledgments

- [anacrolix/torrent](https://github.com/anacrolix/torrent) - Excellent Go torrent library
- [jpillora/cloud-torrent](https://github.com/jpillora/cloud-torrent) - Original inspiration
- [Fiber](https://gofiber.io/) - Fast Go web framework
- [Tailwind CSS](https://tailwindcss.com/) - Utility-first CSS framework
- [Cloudflare](https://cloudflare.com/) - Tunnel and security services
- [CIRCL](https://github.com/cloudflare/circl) - Post-quantum cryptography library
