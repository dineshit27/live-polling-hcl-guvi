# Live Polling System - Deployment & Production Architecture

This document describes the production architecture, environment configuration, container deployment, and operational prerequisites for the Live Polling System.

---

## 1. Production Architecture Overview

The system employs a high-throughput, event-driven decoupled architecture:

```
[ Client Browser ]
        │
        ▼ HTTPS
[ Nginx Reverse Proxy / Ingress ] (:3000 or :443)
        │
        ├──▶ Static SPA Files (React 18 + Vite)
        └──▶ /api/* (HTTP REST & SSE Stream)
                 │
                 ▼ (Keep-Alive, No-Buffering)
        [ Go / Gin API Service ] (:8080)
                 │
                 ├──▶ MongoDB 7.0 (Source of Truth: collections 'users', 'polls', 'votes')
                 │
                 └──▶ Redis 7.0 Pub/Sub (Event Bus: channel 'poll:events:{poll_id}')
                          │
                          ▼ (Goroutine Subscriber)
                      SSE Stream (text/event-stream)
```

---

## 2. Environment Variables & Secrets Management

The application is completely configured via environment variables. See `.env.example` for the full reference.

### Required Environment Variables

| Variable | Description | Local Default | Production Requirement |
| :--- | :--- | :--- | :--- |
| `GIN_MODE` | Gin operational mode | `debug` | `release` |
| `PORT` | Go backend HTTP listen port | `8080` | `8080` (or platform `$PORT`) |
| `JWT_SECRET` | Secret key for signing JWT tokens | dev fallback | **Required**: 32+ character high-entropy secret |
| `MONGO_URI` | MongoDB connection URI | `mongodb://mongodb:27017` | **Required**: Production standalone/Atlas SRV URI |
| `MONGO_DB_NAME` | Target database name | `polling_db` | `polling_db` |
| `REDIS_ADDR` | Redis host:port | `redis:6379` | **Required**: Production Redis endpoint |
| `REDIS_PASSWORD` | Redis authentication password | *(empty)* | Set if Redis AUTH is enabled |
| `CORS_ALLOWED_ORIGINS` | Comma-separated allowed origins | `localhost:3000` | **Required**: Exact HTTPS domain(s), no wildcard `*` |

> **Security Rule**: No secrets (`JWT_SECRET`, database passwords, Redis credentials) are ever prefixed with `VITE_` or embedded into frontend bundles.

---

## 3. Database (MongoDB) Requirements

- **Engine**: MongoDB 7.0+ (Standalone, Replica Set, or MongoDB Atlas).
- **Persistence**: Persistent storage volume mounted to `/data/db` in Docker Compose or managed cloud volume.
- **Indexes**: Automatically verified and initialized by the Go service on startup via `EnsureIndexes`:
  - `users`: `email` (Unique: `uniq_user_email`)
  - `polls`: `owner_id` (`idx_poll_owner`), `created_at` descending (`idx_poll_created_at`)
  - `votes`: Compound `poll_id` + `voter_identifier` (Unique: `uniq_poll_voter`), `poll_id` (`idx_vote_poll_id`)
- **Deduplication**: Strict voter deduplication is guaranteed at the database layer through the compound unique index on `(poll_id, voter_identifier)`.

---

## 4. Realtime Event Bus (Redis) Requirements

- **Engine**: Redis 7.0+ (Redis Standalone, Sentinel, Cluster, or managed services like Redis Cloud / AWS ElastiCache).
- **Pub/Sub Mechanism**: Pub/Sub is leveraged strictly as an ephemeral, low-latency event fan-out bus. Redis is **not** used as a primary persistent datastore.
- **Channels**: Poll-isolated channels using naming convention `poll:{poll_id}:events` with optional global dashboard fan-out on `polls:events`.
- **Subscriber Cleanup**: Goroutine-managed subscribers automatically close Redis subscriptions when client SSE streams disconnect, avoiding connection exhaustion.

---

## 5. Server-Sent Events (SSE) Requirements

- **Headers**: `Content-Type: text/event-stream`, `Cache-Control: no-cache, no-transform`, `Connection: keep-alive`, `X-Accel-Buffering: no`.
- **Proxy Buffering**: Any intermediary reverse proxy (Nginx, Cloudflare, Traefik, AWS ALB) **MUST** have response buffering disabled for `/api/polls/*/events`.
  - In Nginx: `proxy_buffering off;` and `proxy_read_timeout 86400s;`.
- **Keep-Alive Heartbeat**: Built-in 15-second heartbeat comments (`: ping\n\n`) ensure long-lived idle connections are not prematurely severed by stateful firewalls or load balancers.

---

## 6. Local Docker Compose Setup

### Prerequisites
- Docker Engine 24.0+
- Docker Compose v2.20+

### Execution Commands

1. **Configure Environment**:
   ```bash
   cp .env.example .env
   # Edit .env with custom secrets if desired
   ```

2. **Validate Configuration**:
   ```bash
   docker compose config
   ```

3. **Build & Start Containers**:
   ```bash
   docker compose up -d --build
   ```

4. **Verify Container Health**:
   ```bash
   docker compose ps
   curl -i http://localhost:8080/api/health
   curl -i http://localhost:3000/api/health
   ```

5. **Stop Containers**:
   ```bash
   docker compose down -v
   ```

---

## 7. Build Commands

### Backend (Go 1.23)
```bash
cd backend
go mod tidy
CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/api
GOTOOLCHAIN=local go test -v -count=1 ./tests/...
```

### Frontend (React / Vite)
```bash
npm install
npm run lint
npm run build
```

---

## 8. Production Deployment & Rollback Procedures

### Recommended Production Strategy (Option B: Unified Docker Compose on VPS / EC2 / Droplet)
1. **Provision Host**: Ubuntu 22.04 LTS VPS with Docker & Docker Compose plugin installed.
2. **Clone & Configure**:
   ```bash
   git clone <repo-url> /opt/live-polling
   cd /opt/live-polling
   cp .env.example .env
   ```
3. **Configure Production Secrets**:
   Generate a cryptographic 32-byte secret for JWT and set production values in `.env`:
   ```bash
   JWT_SECRET=$(openssl rand -hex 32)
   GIN_MODE=release
   CORS_ALLOWED_ORIGINS=https://polls.yourdomain.com
   ```
4. **Launch Application**:
   ```bash
   docker compose up -d --build
   ```
5. **Verify Running Containers**:
   ```bash
   docker compose ps
   curl -f http://127.0.0.1:8080/api/health
   ```
6. **SSL / Reverse Proxy Setup**:
   Point your domain's DNS A-record to the server IP and configure Certbot / Nginx for TLS:
   ```bash
   sudo certbot --nginx -d polls.yourdomain.com
   ```

### Rollback & Recovery Procedures
- **Restart All Services**:
  ```bash
  docker compose restart
  ```
- **Safe Graceful Stop**:
  ```bash
  docker compose down
  ```
- **Rollback to Previous Image/Commit**:
  ```bash
  git checkout <previous-commit-hash>
  docker compose up -d --build
  ```
- **Inspect Live Logs**:
  ```bash
  docker compose logs -f backend
  docker compose logs -f frontend
  ```

