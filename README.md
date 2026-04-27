<div align="center">

# ContactHub

**A secure, multi-user contact management API built with Go, PostgreSQL, and Docker.**

[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?style=flat&logo=go&logoColor=white)](https://go.dev/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15-336791?style=flat&logo=postgresql&logoColor=white)](https://www.postgresql.org/)
[![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?style=flat&logo=docker&logoColor=white)](https://www.docker.com/)

[Features](#features) · [Quick Start](#quick-start) · [Architecture](#architecture) · [Documentation](#documentation)

</div>

---

## Overview

ContactHub is a backend-driven contact management system with JWT authentication, role-based access control, and a fully Dockerized deployment. Each user manages their own isolated set of contacts with full CRUD operations, soft-delete lifecycles, and automated background cleanup.

---

## Features

- **Authentication** — Signup, login, email verification, and password reset via JWT
- **Role-based access** — User and admin roles with dedicated admin endpoints
- **Contact CRUD** — Create, read, update, soft-delete, restore, and permanently delete
- **Search & filtering** — Case-insensitive search across name, phone, and email with category filtering
- **Soft-delete lifecycle** — Deleted contacts are retained for 30 days, then automatically purged
- **Duplicate detection** — Phone number normalization with per-user uniqueness enforcement
- **Rate limiting** — IP and email-based rate limiting on sensitive endpoints
- **Health monitoring** — `/health` endpoint with database connectivity check
- **Idempotent migrations** — Schema migrations run automatically on every startup, safe to re-run
- **Consistent API** — All endpoints return structured JSON with proper HTTP status codes

---

## Tech Stack

| Layer        | Technology                                           |
| ------------ | ---------------------------------------------------- |
| **Backend**  | Go 1.25, [chi](https://github.com/go-chi/chi) v5    |
| **Database** | PostgreSQL 15                                        |
| **Auth**     | JWT (golang-jwt) + bcrypt                            |
| **Email**    | [Resend](https://resend.com) API                     |
| **Infra**    | Docker, Docker Compose                               |

---

## Architecture

```
                ┌──────────────────┐
 HTTP Request   │   Go Backend     │
 ──────────────▶│   :8080          │
                │                  │
                │  Router          │
                │  ├─ Middleware    │
                │  ├─ Handlers     │
                │  ├─ Services     │
                │  └─ Repository   │
                └────────┬─────────┘
                         │ Docker DNS (host=db)
                ┌────────▼─────────┐
                │  PostgreSQL 15   │
                │  :5432           │
                └──────────────────┘
```

Both services run in Docker Compose on a shared bridge network. The backend resolves the database via Docker's internal DNS using the service name `db`.

---

## Quick Start

### Prerequisites

- [Docker](https://docs.docker.com/get-docker/) (20.10+)
- [Docker Compose](https://docs.docker.com/compose/install/) (v2+)

### Run

```bash
# Clean any previous state
docker compose down -v --remove-orphans

# Build and start
docker compose up --build
```

### Verify

```bash
curl http://localhost:8080/health
```

Expected response:

```json
{
  "status": "ok",
  "db": "connected"
}
```

The backend will be available at `http://localhost:8080`.

---

## Project Structure

```
contact-manager/
├── backend/
│   ├── cmd/api/              # Entry point and route wiring
│   ├── internal/
│   │   ├── config/           # Environment configuration
│   │   ├── database/         # DB connection, retries, migration runner
│   │   ├── handlers/         # HTTP handlers (auth, contacts, admin, health)
│   │   ├── middleware/       # JWT auth, admin guard
│   │   ├── models/           # Data structures
│   │   ├── ratelimit/        # In-memory rate limiter
│   │   ├── repository/       # SQL query layer
│   │   ├── services/         # Business logic
│   │   ├── utils/            # JWT, mailer, response helpers
│   │   └── worker/           # Background cleanup worker
│   ├── migrations/           # Idempotent SQL migration files
│   ├── Dockerfile            # Multi-stage build (Go → Alpine)
│   └── README.md             # ← Detailed backend documentation
├── frontend/                 # React/Next.js frontend (optional)
├── docker-compose.yml        # Backend + PostgreSQL orchestration
└── README.md                 # ← You are here
```

---

## Documentation

For detailed backend architecture, API endpoints, environment variables, and database design, see **[Backend README](./backend/README.md)**.

It covers:

- All 20 API endpoints with example `curl` requests and responses
- Environment variable reference
- Migration system and idempotency guarantees
- Error handling contract
- Internal package structure

---

## License

This project is for educational and portfolio purposes.
