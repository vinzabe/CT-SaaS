# Security Policy

> **Live Demo:** [https://torrent.abejar.net](https://torrent.abejar.net)

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 1.1.x   | :white_check_mark: |
| 1.0.x   | :white_check_mark: |
| < 1.0   | :x:                |

## Reporting a Vulnerability

We take security seriously at CT-SaaS. If you discover a security vulnerability, please follow these steps:

### Do NOT

- Do not open a public GitHub issue
- Do not disclose the vulnerability publicly before it's fixed
- Do not exploit the vulnerability beyond what's necessary to demonstrate it

### Do

1. **Email us directly** at security@abejar.net (or open a private security advisory on GitHub)
2. **Provide details** including:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Any suggested fixes (optional but appreciated)
3. **Allow time** for us to respond (typically within 48 hours) and fix the issue

### What to Expect

- **Acknowledgment**: We'll acknowledge receipt within 48 hours
- **Updates**: We'll keep you informed of our progress
- **Credit**: With your permission, we'll credit you in the security advisory
- **Timeline**: We aim to fix critical vulnerabilities within 7 days

---

## Security Features

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

---

## Security Best Practices for Deployment

### Required for Production

1. **Change Default Credentials**
   ```bash
   # Update demo/admin passwords immediately after deployment
   # Or disable demo accounts entirely in production
   ```

2. **Set a Strong JWT_SECRET**
   ```bash
   # Generate a secure random secret (64+ characters)
   export JWT_SECRET=$(openssl rand -hex 32)
   ```

3. **Use Real SSL Certificates**
   ```bash
   # Replace self-signed certificates with Let's Encrypt or similar
   mkdir -p certs
   cp /etc/letsencrypt/live/yourdomain/fullchain.pem certs/server.crt
   cp /etc/letsencrypt/live/yourdomain/privkey.pem certs/server.key
   ```

4. **Configure Firewall Rules**
   ```bash
   # Only expose frontend ports (7843/7844)
   # Keep API port (7842) internal only
   ufw allow 7843/tcp
   ufw allow 7844/tcp
   ufw deny 7842/tcp  # Block external API access
   ```

### Recommended

5. **Enable Database Encryption**
   - Use encrypted PostgreSQL connections
   - Enable disk encryption for the database volume

6. **Regular Security Updates**
   ```bash
   # Keep dependencies updated
   cd backend && go get -u ./...
   cd frontend && npm update
   ```

7. **Monitor Logs**
   ```bash
   # Check for suspicious activity
   docker logs ct-saas-api --since 24h | grep -i "unauthorized\|failed\|error"
   ```

8. **Backup Strategy**
   - Regular database backups
   - Encrypted backup storage
   - Test restore procedures

---

## Security Architecture

```
                                    [Cloudflare Tunnel]
                                           |
                                    [TLS Termination]
                                           |
                    +----------------------+----------------------+
                    |                                             |
              [HTTPS:7843/7844]                            [HTTPS:7843/7844]
                    |                                             |
              +-----+-----+                               +-------+-------+
              |  Nginx    |                               |    Nginx      |
              |  (HSTS)   |                               |   (HSTS)      |
              +-----------+                               +---------------+
                    |                                             |
              [Reverse Proxy]                            [Static Assets]
                    |
              +-----+-----+
              |  Go API   |
              |  (Fiber)  |
              +-----------+
                    |
         +---------+---------+
         |                   |
   +-----+-----+      +------+------+
   | PostgreSQL|      |    Redis    |
   | (Argon2)  |      | (Sessions)  |
   +-----------+      +-------------+
```

### Data Flow Security

1. **User Authentication**
   - Password -> Argon2id hash -> PostgreSQL
   - Login -> JWT access token (15 min) + refresh token (7 days)
   - Refresh tokens hashed (SHA-256) before storage

2. **API Requests**
   - All requests require valid JWT in Authorization header
   - Rate limiting per user/IP
   - Input validation on all endpoints

3. **File Downloads**
   - Secure token generation (256-bit random)
   - Time-limited tokens (24 hours)
   - Download count limits (10 per token)
   - Path traversal protection

4. **Real-time Updates (SSE)**
   - Token-based authentication via query parameter
   - Connection timeout after 30 minutes
   - Auto-reconnect with backoff

---

## Compliance

CT-SaaS security measures align with:

- **OWASP Top 10** - Protection against common web vulnerabilities
- **NIST Cybersecurity Framework** - Security controls and best practices
- **NIST FIPS 204** - Post-quantum cryptographic standards

---

## Security Contacts

- **Security Issues**: Open a private security advisory on GitHub
- **General Questions**: Open a regular GitHub issue
- **Urgent Issues**: Contact maintainers directly

---

## Changelog

### Security Updates

| Date | Version | Description |
|------|---------|-------------|
| 2026-01-19 | 1.1.0 | Added SSE with token-based auth |
| 2026-01-19 | 1.0.0 | Initial release with PQC support |
