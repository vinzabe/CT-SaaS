# Grant's Torrent

A modern, secure SaaS platform for converting torrent and magnet links to direct HTTP downloads.

Built as a complete rewrite of [jpillora/cloud-torrent](https://github.com/jpillora/cloud-torrent) with modern architecture, full SaaS features, and post-quantum cryptographic security.

## Demo Admin Access

- **Email**: `admin@grants.torrent`
- **Password**: `admin123`

## Features

- **Torrent to Direct Link**: Convert any torrent or magnet link to a streamable HTTP download
- **Post-Quantum Security**: Future-proof encryption using NIST-approved algorithms
- **User Management**: Full authentication with JWT tokens and refresh token rotation
- **Subscription Plans**: Free, Starter, Pro, and Unlimited tiers with usage quotas
- **Admin Panel**: Manage users, view statistics, and monitor the platform
- **Real-time Updates**: Live progress updates via Server-Sent Events
- **Streaming Support**: Direct streaming with Range request support
- **File Management**: Per-file download control and automatic cleanup

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

## Quick Start

### Prerequisites
- Go 1.22+
- Node.js 20+
- Docker & Docker Compose
- PostgreSQL 16 (or use Docker)
- Redis 7 (or use Docker)

### Development Setup

1. **Clone the repository**
```bash
git clone https://github.com/yourusername/freetorrent.git
cd freetorrent
```

2. **Start development databases**
```bash
make dev-db
```

3. **Install dependencies**
```bash
make install
```

4. **Configure environment**
```bash
cp backend/.env.example backend/.env
# Edit backend/.env with your settings
```

5. **Run the backend**
```bash
make dev-backend
```

6. **Run the frontend** (in a new terminal)
```bash
make dev-frontend
```

7. **Access the application**
- Frontend: http://localhost:7843
- Backend API: http://localhost:7842

### Production Deployment

Using Docker Compose:

```bash
# Set production environment variables
export JWT_SECRET="your-secure-secret-key"

# Build and start
docker-compose up -d
```

The application will be available at http://localhost

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
- `GET /api/v1/download/:token` - Download file (public)

### Admin
- `GET /api/v1/admin/users` - List users
- `GET /api/v1/admin/users/:id` - Get user details
- `PATCH /api/v1/admin/users/:id` - Update user
- `DELETE /api/v1/admin/users/:id` - Delete user
- `GET /api/v1/admin/torrents` - List all torrents
- `DELETE /api/v1/admin/torrents/:id` - Delete torrent
- `GET /api/v1/admin/stats` - Platform statistics
- `POST /api/v1/admin/cleanup` - Cleanup expired torrents

### Real-time
- `GET /api/v1/events` - SSE stream for torrent updates

## Subscription Plans

| Plan | Price | Bandwidth | Concurrent | Retention |
|------|-------|-----------|------------|-----------|
| Free | $0/mo | 2 GB/mo | 1 | 24 hours |
| Starter | $5/mo | 50 GB/mo | 3 | 7 days |
| Pro | $15/mo | 500 GB/mo | 10 | 30 days |
| Unlimited | $30/mo | Unlimited | 25 | 90 days |

## Security Features

- **Argon2id**: OWASP-recommended password hashing
- **JWT with Rotation**: Short-lived access tokens with refresh token rotation
- **Rate Limiting**: Per-user and per-IP rate limiting
- **Path Traversal Protection**: Secure file serving
- **CORS Configuration**: Configurable allowed origins
- **Security Headers**: X-Frame-Options, CSP, XSS protection

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | `8080` |
| `ENVIRONMENT` | `development` or `production` | `development` |
| `DATABASE_URL` | PostgreSQL connection string | - |
| `REDIS_URL` | Redis connection string | - |
| `JWT_SECRET` | JWT signing secret | - |
| `JWT_ACCESS_EXPIRY` | Access token expiry (minutes) | `15` |
| `JWT_REFRESH_EXPIRY` | Refresh token expiry (days) | `7` |
| `DOWNLOAD_DIR` | Torrent download directory | `./downloads` |
| `MAX_CONCURRENT` | Max concurrent torrents per user | `10` |
| `TORRENT_PORT` | BitTorrent listen port | `42069` |

## Project Structure

```
freetorrent/
├── backend/
│   ├── cmd/server/         # Application entry point
│   ├── internal/
│   │   ├── auth/           # Authentication logic
│   │   ├── config/         # Configuration
│   │   ├── database/       # Database layer
│   │   ├── handlers/       # HTTP handlers
│   │   ├── middleware/     # HTTP middleware
│   │   ├── models/         # Data models
│   │   └── torrent/        # Torrent engine
│   └── go.mod
├── frontend/
│   ├── src/
│   │   ├── components/     # React components
│   │   ├── hooks/          # Custom hooks
│   │   ├── lib/            # Utilities and API
│   │   ├── pages/          # Page components
│   │   └── types/          # TypeScript types
│   └── package.json
├── docker/
│   ├── Dockerfile.backend
│   ├── Dockerfile.frontend
│   └── nginx.conf
├── docker-compose.yml
├── docker-compose.dev.yml
└── Makefile
```

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the AGPL-3.0 License - see the LICENSE file for details.

## Acknowledgments

- [anacrolix/torrent](https://github.com/anacrolix/torrent) - Excellent Go torrent library
- [jpillora/cloud-torrent](https://github.com/jpillora/cloud-torrent) - Original inspiration
- [Fiber](https://gofiber.io/) - Fast Go web framework
- [Tailwind CSS](https://tailwindcss.com/) - Utility-first CSS framework
