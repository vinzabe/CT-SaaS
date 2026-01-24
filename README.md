<p align="center">
  <!-- Replace with your actual logo -->
  <img src="docs/screenshots/logo-placeholder.png" alt="CT-SaaS Logo" width="200" height="200">
</p>

<h1 align="center">CT-SaaS</h1>

<p align="center">
  <strong>A modern, secure SaaS platform for converting torrent and magnet links to direct HTTP downloads.</strong>
</p>

<p align="center">
  <a href="https://github.com/vinzabe/CT-SaaS/actions/workflows/ci.yml"><img src="https://github.com/vinzabe/CT-SaaS/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://github.com/vinzabe/CT-SaaS/actions/workflows/docker.yml"><img src="https://github.com/vinzabe/CT-SaaS/actions/workflows/docker.yml/badge.svg" alt="Docker Build"></a>
  <img src="https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white" alt="Go Version">
  <img src="https://img.shields.io/badge/Docker-Ready-2496ED?logo=docker&logoColor=white" alt="Docker">
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-AGPL--3.0-blue" alt="License"></a>
</p>

<p align="center">
  <a href="https://torrent.abejar.net"><img src="https://img.shields.io/badge/demo-live-brightgreen?style=for-the-badge" alt="Live Demo"></a>
</p>

---

Built as a complete rewrite of [jpillora/cloud-torrent](https://github.com/jpillora/cloud-torrent) with modern architecture, full SaaS features, and post-quantum cryptographic security.

## Feature Highlights

- :zap: **Instant Conversion** - Transform torrents and magnet links to direct HTTP downloads
- :package: **Auto-ZIP** - Multi-file torrents automatically packaged for easy download
- :lock: **SSL/HTTPS Always On** - Both frontend ports secured with TLS 1.2/1.3
- :shield: **Post-Quantum Security** - Future-proof encryption using NIST-approved ML-DSA-65
- :busts_in_silhouette: **User Management** - Full auth with JWT tokens and refresh rotation
- :credit_card: **Subscription Plans** - Free, Starter, Pro, and Unlimited tiers
- :bar_chart: **Admin Panel** - Manage users, view stats, and monitor the platform
- :satellite: **Real-time Updates** - Live progress via Server-Sent Events (SSE)
- :arrow_forward: **Streaming Support** - Direct streaming with Range request support
- :floppy_disk: **Persistent Storage** - Database and downloads survive restarts

## Architecture

```
                                    +-----------------+
                                    |   Web Browser   |
                                    +--------+--------+
                                             |
                                    HTTPS (7843/7844)
                                             |
+------------------------------------------------------------------------------------+
|                                    Docker Network                                   |
|                                                                                    |
|  +------------------+        +------------------+        +------------------+      |
|  |                  |        |                  |        |                  |      |
|  |   Nginx (web)    +------->+   Go API (api)   +------->+   PostgreSQL     |      |
|  |   :7843/:7844    |  HTTP  |      :7842       |        |   (postgres)     |      |
|  |                  |        |                  |        |                  |      |
|  +------------------+        +--------+---------+        +------------------+      |
|                                       |                                            |
|                                       |                  +------------------+      |
|                                       +----------------->+     Redis        |      |
|                                                          |    (redis)       |      |
|                                                          +------------------+      |
|                                                                                    |
|  +------------------+                                                              |
|  |  Torrent Engine  |  BitTorrent Protocol (TCP/UDP :42069)                       |
|  |  (anacrolix)     +<----------------------------------------------------------+ |
|  +------------------+                                                              |
+------------------------------------------------------------------------------------+
```

```mermaid
graph TB
    subgraph Client
        Browser[Web Browser]
    end
    
    subgraph "Docker Compose Stack"
        Nginx[Nginx<br/>:7843/:7844 HTTPS]
        API[Go API<br/>:7842]
        Torrent[Torrent Engine<br/>:42069]
        PG[(PostgreSQL)]
        Redis[(Redis)]
    end
    
    Browser -->|HTTPS| Nginx
    Nginx -->|HTTP| API
    API --> PG
    API --> Redis
    API --> Torrent
    Torrent -->|BitTorrent| Internet((Internet))
```

## Quick Start

### Prerequisites
- Docker & Docker Compose

### One-Command Deployment

```bash
# Clone the repository
git clone https://github.com/vinzabe/CT-SaaS.git
cd CT-SaaS

# Start CT-SaaS
make up

# Or using docker-compose directly
docker-compose up -d
```

**Access (HTTPS enabled by default):**
- Frontend: https://localhost:7843 or https://localhost:7844
- API: http://localhost:7842

> **Note:** Self-signed SSL certificates are generated automatically. Your browser will show a security warning - this is normal for self-signed certs.

### Docker Compose Configuration

```yaml
# docker-compose.yml (simplified)
version: '3.8'
services:
  api:
    build:
      context: .
      dockerfile: docker/Dockerfile.backend
    ports:
      - "7842:7842"
      - "42069:42069"
    environment:
      - DATABASE_URL=postgres://user:pass@postgres:5432/db
      - REDIS_URL=redis://redis:6379
      - JWT_SECRET=your-secret-here
    depends_on:
      - postgres
      - redis

  web:
    build:
      context: .
      dockerfile: docker/Dockerfile.frontend
    ports:
      - "7843:7843"
      - "7844:7844"
    depends_on:
      - api

  postgres:
    image: postgres:16-alpine

  redis:
    image: redis:7-alpine
```

### Common Commands

```bash
# Start production (Docker)
make up

# Stop all containers
make down

# Stop and wipe all data
./stop.sh clean

# View logs
docker logs ct-saas-api
docker logs ct-saas-web
```

## Demo Accounts

| Account | Email | Password | Notes |
|---------|-------|----------|-------|
| **Admin** | `admin@ct.saas` | `admin123` | Full access, admin panel |
| **Demo** | `demo@ct.saas` | `demo123` | Restricted, 24hr retention |

> **Try the live demo:** [https://torrent.abejar.net](https://torrent.abejar.net)

## Screenshots

<p align="center">
  <img src="screenshots/dashboard.png" alt="CT-SaaS Dashboard" width="800">
</p>

<details>
<summary>More Screenshots</summary>

### Landing Page
<p align="center">
  <img src="screenshots/landing.png" alt="Landing Page" width="800">
</p>

### Admin Panel
<p align="center">
  <img src="screenshots/admin.png" alt="Admin Panel" width="800">
</p>

### Add Torrent
<p align="center">
  <img src="screenshots/add-torrent.png" alt="Add Torrent Modal" width="600">
</p>

</details>

## Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `PORT` | Backend server port | `7842` | No |
| `ENVIRONMENT` | `development` or `production` | `production` | No |
| `DATABASE_URL` | PostgreSQL connection string | Docker internal | Yes |
| `REDIS_URL` | Redis connection string | Docker internal | Yes |
| `JWT_SECRET` | JWT signing secret (64+ chars recommended) | Auto-generated | **Yes (prod)** |
| `JWT_ACCESS_EXPIRY` | Access token expiry (minutes) | `15` | No |
| `JWT_REFRESH_EXPIRY` | Refresh token expiry (days) | `7` | No |
| `DOWNLOAD_DIR` | Torrent download directory | `/downloads` | No |
| `TORRENT_PORT` | BitTorrent listen port | `42069` | No |
| `MAX_CONCURRENT` | Max concurrent torrents | `10` | No |
| `STRIPE_SECRET_KEY` | Stripe API key for payments | - | No |
| `STRIPE_WEBHOOK_KEY` | Stripe webhook secret | - | No |

## API Endpoints

### Authentication

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/v1/auth/register` | Create new account |
| `POST` | `/api/v1/auth/login` | Login and get tokens |
| `POST` | `/api/v1/auth/refresh` | Refresh access token |
| `POST` | `/api/v1/auth/logout` | Logout and invalidate tokens |
| `GET` | `/api/v1/auth/me` | Get current user info |

### Torrents

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/v1/torrents` | Add torrent (magnet or URL) |
| `POST` | `/api/v1/torrents/upload` | Upload .torrent file |
| `GET` | `/api/v1/torrents` | List user's torrents |
| `GET` | `/api/v1/torrents/:id` | Get torrent details |
| `DELETE` | `/api/v1/torrents/:id` | Delete torrent |
| `POST` | `/api/v1/torrents/:id/pause` | Pause download |
| `POST` | `/api/v1/torrents/:id/resume` | Resume download |
| `POST` | `/api/v1/torrents/:id/token` | Generate download token |

### Real-time Events (SSE)

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/v1/events?token=<jwt>` | Subscribe to torrent updates |
| `GET` | `/api/v1/admin/events?token=<jwt>` | Subscribe to all updates (admin) |

**SSE Events:**
- `connected` - Connection established
- `torrents` - Torrent status updates (progress, speed, peers)
- `heartbeat` - Keep-alive signal
- `timeout` - Connection timeout

### Downloads

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/v1/download/:token` | Download file (token-authenticated) |

### Admin

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/v1/admin/users` | List all users |
| `GET` | `/api/v1/admin/users/:id` | Get user details |
| `PATCH` | `/api/v1/admin/users/:id` | Update user |
| `DELETE` | `/api/v1/admin/users/:id` | Delete user |
| `GET` | `/api/v1/admin/torrents` | List all torrents |
| `DELETE` | `/api/v1/admin/torrents/:id` | Delete any torrent |
| `GET` | `/api/v1/admin/stats` | Platform statistics |
| `POST` | `/api/v1/admin/cleanup` | Cleanup expired torrents |

## Subscription Plans

| Plan | Price | Bandwidth | Concurrent | Retention |
|------|-------|-----------|------------|-----------|
| Free | $0/mo | 2 GB/mo | 1 | 24 hours |
| Starter | $5/mo | 50 GB/mo | 3 | 7 days |
| Pro | $15/mo | 500 GB/mo | 10 | 30 days |
| Unlimited | $30/mo | Unlimited | 25 | 90 days |

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

## Security

CT-SaaS implements multiple layers of security:

| Feature | Implementation |
|---------|----------------|
| Password Hashing | Argon2id (OWASP recommended) |
| Access Tokens | JWT with 15-minute expiry |
| Refresh Tokens | Secure random, SHA-256 hashed |
| Post-Quantum | ML-DSA-65 (NIST FIPS 204) |
| TLS | 1.2/1.3 with modern ciphers |
| Rate Limiting | 100 requests/minute per user |

See [SECURITY.md](SECURITY.md) for full details.

## Roadmap

- [ ] **Multi-language support** - i18n for UI
- [ ] **Telegram bot integration** - Add torrents via Telegram
- [ ] **WebTorrent support** - Browser-based seeding
- [ ] **S3 storage backend** - Store downloads in cloud storage
- [ ] **OAuth providers** - Login with Google, GitHub
- [ ] **Mobile app** - React Native companion app
- [ ] **API rate limiting tiers** - Per-plan API limits
- [ ] **Webhook notifications** - Notify on torrent completion
- [ ] **Torrent search** - Integrated search providers
- [ ] **Bandwidth scheduling** - Time-based speed limits

## Development

### Prerequisites
- Go 1.22+
- Node.js 20+
- Docker & Docker Compose

### Local Development

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
- Frontend: http://localhost:5173
- Backend API: http://localhost:7842

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
├── docs/
│   └── screenshots/        # Documentation screenshots
├── docker-compose.yml
├── docker-compose.dev.yml
├── Makefile
├── start.sh
└── stop.sh
```

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

This project is licensed under the AGPL-3.0 License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [anacrolix/torrent](https://github.com/anacrolix/torrent) - Excellent Go torrent library
- [jpillora/cloud-torrent](https://github.com/jpillora/cloud-torrent) - Original inspiration
- [Fiber](https://gofiber.io/) - Fast Go web framework
- [Tailwind CSS](https://tailwindcss.com/) - Utility-first CSS framework
- [Cloudflare CIRCL](https://github.com/cloudflare/circl) - Post-quantum cryptography library
