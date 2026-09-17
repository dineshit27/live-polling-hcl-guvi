<img src="images/Project Banner.png" alt="Live Polling Platform Banner" width="100%">

<div align="center">

# 📊 Live Polling Platform

### Create polls. Share them. Watch votes happen live.

A full-stack real-time polling platform built with **React, Go/Gin, MongoDB, Redis Pub/Sub, and Server-Sent Events (SSE)**.

Build a poll, share it with anyone, collect votes, and watch the results update instantly across connected clients — without refreshing the page.

<img src="https://img.shields.io/badge/Status-Production_Ready-00C853?style=flat-square">
<img src="https://img.shields.io/badge/Version-1.0.0-6366F1?style=flat-square">
<img src="https://img.shields.io/badge/React-19-61DAFB?style=flat-square&logo=react&logoColor=black">
<img src="https://img.shields.io/badge/Go-1.23-00ADD8?style=flat-square&logo=go&logoColor=white">
<img src="https://img.shields.io/badge/Gin-1.10-008ECF?style=flat-square">
<img src="https://img.shields.io/badge/MongoDB-7.0-47A248?style=flat-square&logo=mongodb&logoColor=white">
<img src="https://img.shields.io/badge/Redis-7.x-DC382D?style=flat-square&logo=redis&logoColor=white">
<img src="https://img.shields.io/badge/SSE-Realtime-7C3AED?style=flat-square">

<br>

<img src="https://img.shields.io/badge/Tests-32%2F32_Passing-00C853?style=flat-square">
<img src="https://img.shields.io/badge/Redis_Pub%2FSub-Verified-00C853?style=flat-square">
<img src="https://img.shields.io/badge/Docker-Ready-2496ED?style=flat-square&logo=docker&logoColor=white">

</div>

---

## 🚀 Introduction

**Live Polling Platform** is a real-time web application designed for creating, sharing, voting on, and monitoring polls.

The main engineering challenge is not simply storing votes — it is delivering **live vote updates to every connected client without page refreshes or client-side polling**.

The application uses:

- **React** for the interactive frontend
- **Go + Gin** for the backend API
- **MongoDB** as the persistent source of truth
- **Redis Pub/Sub** as the real-time event transport
- **Server-Sent Events (SSE)** to stream events from Go to connected browsers

When a user votes, the event follows this path:

```text
React Client ➞ Go/Gin API ➞ MongoDB ➞ Redis Pub/Sub ➞ Go Realtime Subscriber ➞ SSE ➞ Connected React Clients
```

No page refresh | No fake timers | No frontend polling.

---

## ✨ Features

| Feature | Description |
|---|---|
| 🔐 Authentication | Signup, login, logout and session restoration |
| 🔑 JWT Security | Token-based authentication with server-side validation |
| 🔒 Password Security | Passwords protected using bcrypt |
| 🗳️ Poll Creation | Create polls with dynamic options |
| 👥 Public Voting | Anyone can access a shared poll and vote |
| ⚡ Live Results | Vote counts update instantly without refresh |
| 📡 Redis Pub/Sub | Real vote events are distributed through Redis |
| 📲 SSE Streaming | Browser clients receive live server events |
| 🚫 Duplicate Vote Protection | Database-level unique constraints prevent repeated voting |
| 📊 Live Percentages | Results and percentages update automatically |
| 👤 My Polls | Poll owners can view and manage their polls |
| 🔐 Ownership Control | Only the poll creator can manage their poll |
| ⛔ Poll Closing | Owners can close polls and stop new votes |
| 🗑️ Poll Deletion | Owners can delete their polls |
| 🔎 Search | Search polls by title and description |
| ↕️ Sorting | Sort by newest or most voted |
| 🟢 Connection Status | Connecting, connected, reconnecting and disconnected states |
| ♻️ Reconnection | SSE automatically attempts to reconnect |
| 📱 Responsive UI | Optimized for desktop, tablet and mobile |
| ♿ Accessible UI | Keyboard focus, readable states and accessible interactions |
| 🧪 Automated Testing | Backend unit/integration coverage |
| 🐳 Docker Ready | MongoDB, Redis, backend and frontend container configuration |

---

## 🎯 Core Problem Solved

Traditional polling applications often require clients to repeatedly request the server:

```text
Browser → GET results
Browser → GET results
Browser → GET results
Browser → GET results
```

This application uses an event-driven architecture instead:

```text
User Vote
   ↓
Go API
   ↓
MongoDB
   ↓
Redis Pub/Sub
   ↓
Go Subscriber
   ↓
SSE
   ↓
All Connected Browsers
```

This allows connected users to see the updated result immediately.

---

# 🏗️ Architecture

```mermaid
flowchart TB

    U1["👤 Voter / Poll Viewer"]
    U2["👤 Poll Owner"]

    FE["⚛️ React Frontend<br/>TypeScript + Tailwind"]

    API["🚀 Go + Gin API"]

    AUTH["🔐 JWT + bcrypt<br/>Authentication"]

    MONGO[("🍃 MongoDB<br/>Source of Truth")]

    REDIS[("🔴 Redis<br/>Pub/Sub")]

    SSE["📡 Go SSE Stream"]

    C1["🌐 Connected Client A"]
    C2["🌐 Connected Client B"]
    C3["🌐 Connected Client N"]

    U1 --> FE
    U2 --> FE

    FE --> API

    API --> AUTH
    API --> MONGO

    API --> REDIS
    REDIS --> SSE

    SSE --> C1
    SSE --> C2
    SSE --> C3
```

---

## 🔄 Real-Time Vote Flow

```mermaid
sequenceDiagram

    participant A as Client A
    participant API as Go/Gin API
    participant DB as MongoDB
    participant R as Redis Pub/Sub
    participant S as Go SSE
    participant B as Client B

    B->>S: Connect to /api/polls/:id/events
    S->>DB: Load current poll state
    DB-->>S: Poll snapshot
    S-->>B: POLL_SNAPSHOT

    A->>API: POST /api/polls/:id/vote
    API->>DB: Validate + persist vote
    DB-->>API: Vote persisted

    API->>DB: Atomic $inc count
    DB-->>API: Updated counts

    API->>R: Publish VOTE_CAST
    R-->>S: Vote event

    S-->>B: SSE VOTE_CAST
    B->>B: Update UI

    Note over A,B: No refresh and no frontend polling
```

---

# 📡 SSE Lifecycle

Each connected poll viewer establishes a Server-Sent Events stream.

```text
CONNECT
   ↓
POLL_SNAPSHOT
   ↓
CONNECTED
   ↓
VOTE_CAST
   ↓
VOTE_CAST
   ↓
POLL_STATUS
   ↓
HEARTBEAT
   ↓
DISCONNECT
   ↓
CLEANUP
```

The frontend exposes connection states:

```text
🟢 Connected
🟡 Connecting
🟠 Reconnecting
🔴 Disconnected
```

The backend also performs cleanup when the request context is cancelled.

---

# 🔐 Security Architecture

```mermaid
flowchart LR

    CLIENT["React Client"]

    TOKEN["JWT Bearer Token"]

    AUTH["Auth Middleware"]

    CLAIM["Verified user_id Claim"]

    OWNER["Ownership Check"]

    API["Protected API"]

    DB[("MongoDB")]

    CLIENT --> TOKEN
    TOKEN --> AUTH
    AUTH --> CLAIM
    CLAIM --> OWNER
    OWNER --> API
    API --> DB
```

### Security measures

- JWT-based authentication
- bcrypt password hashing
- Server-side token validation
- Server-derived ownership
- Generic invalid-login responses
- Request validation
- Duplicate-vote database constraints
- Closed-poll validation
- No frontend JWT secrets
- No sensitive information in public SSE payloads
- No `dangerouslySetInnerHTML`
- CORS configured through environment variables
- Relative `/api/*` requests
- Graceful Redis cleanup
- Standardized API error responses

---

# 📁 Project Structure

```text
live-polling-platform/
│
├── backend/
│   ├── cmd/
│   │   └── api/
│   │       └── main.go
│   │
│   ├── config/
│   ├── handlers/
│   ├── middleware/
│   ├── models/
│   ├── realtime/
│   ├── repositories/
│   ├── routes/
│   ├── services/
│   ├── tests/
│   │
│   ├── Dockerfile
│   ├── go.mod
│   └── go.sum
│
├── src/
│   ├── components/
│   ├── hooks/
│   ├── lib/
│   ├── services/
│   ├── types.ts
│   ├── App.tsx
│   └── main.tsx
│
├── public/
│
├── images/
│   └── Project Banner.png
│
├── Dockerfile
├── docker-compose.yml
├── nginx.conf
├── package.json
├── tsconfig.json
├── vite.config.ts
├── .env.example
├── .gitignore
└── README.md
```

---

# 🛠️ Tech Stack

## Frontend

<img src="https://img.shields.io/badge/React-19-61DAFB?style=for-the-badge&logo=react&logoColor=black"> <img src="https://img.shields.io/badge/TypeScript-5.x-3178C6?style=for-the-badge&logo=typescript&logoColor=white"> <img src="https://img.shields.io/badge/Tailwind_CSS-3.x-06B6D4?style=for-the-badge&logo=tailwindcss&logoColor=white"> <img src="https://img.shields.io/badge/Vite-7.x-646CFF?style=for-the-badge&logo=vite&logoColor=white">

## Backend

<img src="https://img.shields.io/badge/Go-1.23-00ADD8?style=for-the-badge&logo=go&logoColor=white"> <img src="https://img.shields.io/badge/Gin-1.10-008ECF?style=for-the-badge"> <img src="https://img.shields.io/badge/JWT-Authentication-000000?style=for-the-badge"> <img src="https://img.shields.io/badge/bcrypt-Password_Hashing-444444?style=for-the-badge">

## Database & Realtime

<img src="https://img.shields.io/badge/MongoDB-7.0-47A248?style=for-the-badge&logo=mongodb&logoColor=white"> <img src="https://img.shields.io/badge/Redis-7.x-DC382D?style=for-the-badge&logo=redis&logoColor=white"> <img src="https://img.shields.io/badge/Server--Sent_Events-SSE-7C3AED?style=for-the-badge">

## Infrastructure

<img src="https://img.shields.io/badge/Docker-Ready-2496ED?style=for-the-badge&logo=docker&logoColor=white"> <img src="https://img.shields.io/badge/Nginx-Reverse_Proxy-009639?style=for-the-badge&logo=nginx&logoColor=white"> <img src="https://img.shields.io/badge/HTTPS-Production-00C853?style=for-the-badge">

---

# 🎨 UI / UX Design

The interface uses an **Obsidian Editorial** visual direction instead of a generic dashboard template.

### Design characteristics

- Deep midnight slate foundation
- Electric cyan realtime accents
- Signal emerald live states
- Crisp editorial typography
- Plus Jakarta Sans for UI and display text
- JetBrains Mono for counters and telemetry
- Subtle borders
- High-contrast controls
- Smooth progress transitions
- Responsive layouts
- Keyboard-friendly interactions
- Live connection indicators
- Toast-based feedback
- Clear empty and error states

---

## 🖥️ Main Screens

### Landing Page

The landing experience communicates the core product immediately:

```text
Create polls.
Share them.
Watch votes live.
```

It also includes an interactive live-result preview to visually communicate the realtime capability.

---

### Polls Workspace

```text
┌───────────────────────────────────────────────┐
│ Search polls...              Sort: Newest     │
├───────────────────────────────────────────────┤
│ All   Active   Closed   My Polls              │
├───────────────────────────────────────────────┤
│                                               │
│ Poll Card                                     │
│ Live • 24 Votes                               │
│ ███████████████████░░░                        │
│                                               │
└───────────────────────────────────────────────┘
```

---

### Poll Studio

The poll creation interface uses a dual-column layout:

```text
┌───────────────────────┬───────────────────────┐
│                       │                       │
│    Poll Form          │    Live Preview       │
│                       │                       │
│    Title              │    Question           │
│    Description        │    Option A           │
│    Options            │    Option B           │
│                       │    Option C           │
│    Create Poll        │                       │
│                       │                       │
└───────────────────────┴───────────────────────┘
```

---

### Live Results

Results update through SSE:

```text
What should we build next?

React
███████████████████████  58%

Go
███████████████          32%

Rust
█████                    10%

🟢 LIVE
42 participants
```

---

# 📸 Screenshots

> Add your final screenshots inside the `images/` directory and update the filenames below.

<p align="center">
  <img src="images/landing.png" alt="Landing Page" width="48%">
  <img src="images/dashboard.png" alt="Poll Dashboard" width="48%">
</p>

<p align="center">
  <img src="images/create-poll.png" alt="Create Poll" width="48%">
  <img src="images/live-results.png" alt="Live Results" width="48%">
</p>

<p align="center">
  <img src="images/auth.png" alt="Authentication" width="48%">
  <img src="images/mobile.png" alt="Mobile Responsive UI" width="48%">
</p>

---

# 🔌 API Overview

## Authentication

| Method | Endpoint | Purpose |
|---|---|---|
| POST | `/api/auth/signup` | Create account |
| POST | `/api/auth/login` | Authenticate user |
| GET | `/api/auth/me` | Get authenticated profile |

---

## Polls

| Method | Endpoint | Purpose |
|---|---|---|
| POST | `/api/polls` | Create poll |
| GET | `/api/polls` | List active polls |
| GET | `/api/polls/:id` | Get poll |
| GET | `/api/polls/mine` | Get owned polls |
| PUT | `/api/polls/:id` | Update poll |
| POST | `/api/polls/:id/close` | Close poll |
| DELETE | `/api/polls/:id` | Delete poll |

---

## Voting

| Method | Endpoint | Purpose |
|---|---|---|
| POST | `/api/polls/:id/vote` | Cast vote |

---

## Realtime

| Method | Endpoint | Purpose |
|---|---|---|
| GET | `/api/polls/:id/events` | SSE realtime stream |
| GET | `/api/polls/:id/stream` | SSE stream alias |

---

## Health

```text
GET /api/health
```

Expected healthy state:

```json
{
  "status": "healthy"
}
```

The health check also verifies MongoDB and Redis connectivity.

---

# 🧪 Testing

The project was tested at both unit/integration and end-to-end levels.

## Backend

```bash
cd backend
GOTOOLCHAIN=local go test -v -count=1 ./tests
```

Result:

```text
32/32 tests passing
```

---

## Frontend Type Checking

```bash
npm run lint
```

Result:

```text
0 TypeScript errors
```

---

## Production Build

```bash
npm run build
```

Result:

```text
Vite production build successful
```

---

# ✅ Functional Test Matrix

| Test | Expected Result | Status |
|---|---|---|
| Signup | Account created | ✅ |
| Duplicate signup | 409 response | ✅ |
| Valid login | JWT returned | ✅ |
| Wrong password | 401 response | ✅ |
| `/me` with valid token | Profile returned | ✅ |
| `/me` without token | 401 response | ✅ |
| Create poll | Poll persisted | ✅ |
| Fetch poll | Poll returned | ✅ |
| Valid vote | Vote persisted | ✅ |
| Duplicate vote | 409 response | ✅ |
| Invalid option | 400 response | ✅ |
| Nonexistent poll | 404 response | ✅ |
| Closed poll vote | 400 response | ✅ |
| Owner close | Poll closed | ✅ |
| Non-owner close | 403 response | ✅ |
| Owner delete | Poll deleted | ✅ |
| Non-owner delete | 403 response | ✅ |
| SSE snapshot | Initial state received | ✅ |
| SSE live vote | Event received | ✅ |
| Poll isolation | No cross-poll events | ✅ |
| Multiple subscribers | All receive event | ✅ |
| Redis Pub/Sub | Event delivered | ✅ |
| Reconnect | Client retries connection | ✅ |
| MongoDB persistence | Data survives request | ✅ |
| Redis disconnect cleanup | Resources released | ✅ |

---

# ⚡ Realtime Verification

The most important end-to-end scenario was tested using two clients.

### Client A

```text
Open Poll
   ↓
Vote for Option A
```

### Backend

```text
Vote Request
    ↓
Validation
    ↓
MongoDB Vote Insert
    ↓
MongoDB Atomic $inc
    ↓
Redis Publish
```

### Client B

```text
Redis Subscriber
      ↓
Go SSE Handler
      ↓
EventSource
      ↓
Live Result Update
```

### Result

```text
Client A votes
      ↓
MongoDB updated
      ↓
Redis event published
      ↓
Go receives event
      ↓
SSE sends event
      ↓
Client B updates
      ↓
NO PAGE REFRESH
```

---

# 🛡️ Duplicate Vote Protection

Duplicate voting is enforced at the database level.

The application creates a unique constraint using:

```text
poll_id + voter_identifier
```

This means even if multiple requests reach the backend simultaneously, MongoDB remains the final enforcement layer.

The backend also validates:

- Poll existence
- Poll status
- Option ID
- Voter identity
- Authentication state where applicable

---

# 🔄 Failure & Recovery

The application handles several failure scenarios.

### Redis unavailable

MongoDB remains the persistent source of truth.

A vote that has already been successfully persisted is not lost simply because realtime delivery is temporarily unavailable.

---

### SSE disconnect

The frontend detects the connection loss:

```text
CONNECTED
   ↓
DISCONNECTED
   ↓
RECONNECTING
   ↓
CONNECTED
```

The frontend can reconcile the latest poll state after reconnecting.

---

### Malformed Redis event

Realtime subscribers safely handle malformed messages instead of allowing an invalid event to crash the stream.

---

### Poll isolation

Each poll uses its own Redis channel:

```text
poll:A:events
poll:B:events
poll:C:events
```

A vote from Poll A does not update Poll B.

---

# 🌐 Production Architecture

```mermaid
flowchart TB

    USER["🌐 User Browser"]

    HTTPS["🔒 HTTPS / TLS"]

    NGINX["Nginx<br/>Reverse Proxy"]

    REACT["React Static Application"]

    GO["Go + Gin API"]

    MONGO[("MongoDB<br/>Persistent Database")]

    REDIS[("Redis<br/>Pub/Sub")]

    SSE["Go SSE Layer"]

    USER --> HTTPS
    HTTPS --> NGINX

    NGINX --> REACT
    NGINX --> GO

    GO --> MONGO
    GO --> REDIS
    REDIS --> SSE

    SSE --> GO
    GO --> NGINX
    NGINX --> USER
```

---

# 🐳 Docker

The repository contains Docker configuration for the full stack.

## Services

```text
mongodb
redis
backend
frontend
```

---

## Start

```bash
docker compose up -d --build
```

---

## Check containers

```bash
docker compose ps
```

---

## View logs

```bash
docker compose logs -f backend
```

---

## Stop

```bash
docker compose down
```

---

## Stop and remove volumes

```bash
docker compose down -v
```

> Removing volumes deletes local MongoDB and Redis persisted data.

---

# 🚀 Getting Started

## Prerequisites

Install:

- Node.js 20+
- Go 1.23+
- MongoDB 7+
- Redis 7+
- Git
- Docker Desktop — optional

---

## 1. Clone

```bash
git clone YOUR_GITHUB_REPOSITORY_URL

cd live-polling-platform
```

---

## 2. Configure Environment

```bash
cp .env.example .env
```

Update the required values.

---

## 3. Start MongoDB & Redis

If using Docker:

```bash
docker compose up -d mongodb redis
```

---

## 4. Start Backend

```bash
cd backend

go mod download

go run ./cmd/api
```

Backend:

```text
http://localhost:8080
```

---

## 5. Start Frontend

From the project root:

```bash
npm install
npm run dev
```

Frontend:

```text
http://localhost:3000
```

---

# 🔍 Health Check

Open:

```text
/api/health
```

or:

```bash
curl http://localhost:3000/api/health
```

The application should report a healthy backend with MongoDB and Redis connected.

---

# 📊 Engineering Metrics

| Metric | Result |
|---|---:|
| Backend tests | 32/32 |
| TypeScript errors | 0 |
| Frontend production build | Passing |
| Redis Pub/Sub | Verified |
| MongoDB persistence | Verified |
| SSE realtime | Verified |
| Duplicate vote protection | Verified |
| Poll isolation | Verified |
| JWT authentication | Verified |
| Ownership authorization | Verified |
| Mobile responsive UI | Verified |
| Docker configuration | Ready |

---

# 🧱 Engineering Decisions

## MongoDB as Source of Truth

MongoDB stores:

- Users
- Polls
- Votes
- Vote counts

Redis is intentionally not used as the permanent database.

---

## Redis Pub/Sub for Realtime

Redis is responsible for distributing events between backend realtime subscribers.

This makes the realtime layer independent from the browser.

---

## SSE Instead of WebSockets

Server-Sent Events were selected because the communication pattern is primarily:

```text
Server → Browser
```

The browser sends votes through normal HTTP requests while the server streams updates through SSE.

---

## Atomic Vote Counts

MongoDB `$inc` is used to update option counts atomically.

The vote persistence and count update flow also includes consistency protection to avoid leaving invalid vote state behind.

---

## Server-Side Ownership

The frontend does not decide whether a user owns a poll.

The backend extracts the authenticated identity from the verified JWT and performs the ownership check server-side.

---

# 🤖 AI-Assisted Development

AI tools were used during development as development assistance rather than as a replacement for understanding the architecture.

AI assistance was used for areas such as:

- Boilerplate generation
- Code organization
- UI iteration
- Test generation
- Debugging
- Documentation
- Realtime architecture review
- Security review
- Error handling improvements

The core architecture was intentionally maintained around the required stack:

```text
React
   ↓
Go / Gin
   ↓
MongoDB
   +
Redis Pub/Sub
   ↓
SSE
```

The implementation was manually verified through:

- Backend tests
- Frontend type checking
- Production builds
- Health checks
- Authentication testing
- Voting tests
- Duplicate vote tests
- Ownership tests
- Two-client realtime testing
- Redis channel isolation testing

---

# 🧠 What I Learned

Building this project required understanding more than just the UI.

### 1. Event-driven architecture

A vote is not simply a database update.

It becomes an event that needs to reach all relevant connected clients.

### 2. Persistence vs realtime

MongoDB answers:

> What is the current truth?

Redis answers:

> Who should receive this event?

### 3. SSE lifecycle management

Long-lived connections require:

- Heartbeats
- Cleanup
- Reconnection
- Context cancellation
- Subscriber management

### 4. Database-level guarantees

Client-side duplicate prevention is not sufficient.

The database must enforce uniqueness.

### 5. Server-side authorization

The browser cannot be trusted to determine ownership.

The backend must derive identity from a verified token.

---

# 🧪 Evaluator Test Scenario

A simple evaluator can verify the core realtime feature with two browser windows.

### Window A

1. Open the application.
2. Login or create an account.
3. Create a poll.
4. Open the poll.

### Window B

1. Open the same poll.
2. Keep the results visible.

### Then

Vote from Window A.

Expected:

```text
Window A
   ↓
Go API
   ↓
MongoDB
   ↓
Redis Pub/Sub
   ↓
Go SSE
   ↓
Window B
```

Window B should update automatically.

### Important

Do **not** refresh Window B.

If the result changes without refreshing, the realtime architecture is working.

---

# 📚 Resources

<div align="center">

<a href="YOUR_LIVE_URL">
  <img src="https://img.shields.io/badge/🚀_Live_Demo-Open-00C853?style=for-the-badge">
</a>

<a href="YOUR_GITHUB_REPOSITORY_URL">
  <img src="https://img.shields.io/badge/💻_Source_Code-GitHub-181717?style=for-the-badge&logo=github">
</a>

<a href="YOUR_DEMO_VIDEO_URL">
  <img src="https://img.shields.io/badge/🎥_Demo_Video-Watch-red?style=for-the-badge&logo=youtube">
</a>

</div>

---

# 🗺️ Development Roadmap

```text
[✓] Architecture
       ↓
[✓] React + Go/Gin scaffolding
       ↓
[✓] MongoDB persistence
       ↓
[✓] Authentication
       ↓
[✓] Poll CRUD
       ↓
[✓] Voting + deduplication
       ↓
[✓] Redis Pub/Sub
       ↓
[✓] Go SSE
       ↓
[✓] React realtime UI
       ↓
[✓] Functional regression
       ↓
[✓] UI/UX redesign
       ↓
[✓] Security audit
       ↓
[✓] Production configuration
       ↓
[ ] Final public deployment
       ↓
[ ] Final demo + submission
```

> Update the final two items after completing the public deployment and submission.

---

# 🤝 Contributing

Contributions, suggestions and improvements are welcome.

### Fork the repository

```bash
git fork YOUR_GITHUB_REPOSITORY_URL
```

### Create a branch

```bash
git checkout -b feature/your-feature
```

### Make your changes

```bash
git add .
git commit -m "feat: add your feature"
```

### Push

```bash
git push origin feature/your-feature
```

Then open a Pull Request.

---

# 📄 License

If this repository contains a license file, the project is distributed under the license specified there.

If you intend to use the MIT License, add a `LICENSE` file before changing this section to:

```text
MIT License
```

---

# 👨‍💻 Developer

<div align="center">

## Dinesh M

B.Tech Information Technology  
Full Stack Developer • Software Developer • Builder

<a href="YOUR_GITHUB_REPOSITORY_URL">
  <img src="https://img.shields.io/badge/GitHub-181717?style=for-the-badge&logo=github&logoColor=white">
</a>

<a href="YOUR_LINKEDIN_URL">
  <img src="https://img.shields.io/badge/LinkedIn-0A66C2?style=for-the-badge&logo=linkedin&logoColor=white">
</a>

<a href="YOUR_PORTFOLIO_URL">
  <img src="https://img.shields.io/badge/Portfolio-Visit-6366F1?style=for-the-badge">
</a>

</div>

---

# ⭐ Support

If you found this project useful, consider giving the repository a ⭐.

It helps the project get noticed and encourages further development.

---

<div align="center">

### Built with React, Go, MongoDB, Redis and a lot of debugging.

**Create polls. Share them. Watch votes live.**

<br>

Made with ❤️ by **Dinesh M**

</div>
