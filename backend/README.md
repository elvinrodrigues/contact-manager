# ContactHub — Backend API

A production-grade REST API for managing personal contacts, built with Go and PostgreSQL. Features JWT-based authentication, role-based access control, email verification, soft-delete with automated cleanup, and a fully Dockerized deployment.

## Overview

ContactHub is a backend service that provides secure, multi-user contact management through a JSON REST API. Each user manages their own isolated set of contacts with full CRUD operations, search, soft-delete/restore workflows, and automatic purging of expired records.

**Key capabilities:**

- JWT authentication with email verification and password reset
- Per-user contact isolation with soft-delete lifecycle
- Role-based access control (user / admin)
- Background worker for automated cleanup of expired contacts
- Idempotent database migrations that run on every startup
- Production health checks with database connectivity verification
- Rate limiting on sensitive endpoints (password reset)

## Tech Stack

| Layer          | Technology                                                   |
| -------------- | ------------------------------------------------------------ |
| **Language**   | Go 1.25                                                      |
| **Router**     | [chi](https://github.com/go-chi/chi) v5                     |
| **Database**   | PostgreSQL 15                                                |
| **Auth**       | JWT ([golang-jwt](https://github.com/golang-jwt/jwt)) + bcrypt |
| **Email**      | [Resend](https://resend.com) API                             |
| **Infra**      | Docker, Docker Compose                                       |

## Features

### Authentication & Authorization
- User signup with email verification flow
- Login with JWT token issuance
- Password reset via email with secure token hashing
- Rate limiting on forgot-password (per-IP and per-email)
- Timing-attack mitigation on account enumeration
- Admin role with dedicated management endpoints

### Contact Management
- Create, read, update, delete contacts
- Soft-delete with 30-day retention before permanent purge
- Restore soft-deleted contacts
- Search across name, phone, and email
- Filter by category
- Paginated list endpoints with total counts
- Per-user phone uniqueness (scoped to active contacts)

### Infrastructure
- Health check endpoint with database ping (2s timeout)
- Background cleanup worker (configurable interval)
- Automatic retry on database connection (10 attempts, 2s apart)
- Consistent JSON response envelope on all endpoints
- CORS configuration for frontend integration

## Architecture Overview

```
┌───────────────┐         ┌──────────────────┐
│  HTTP Client  │────────▶│  Go Backend      │
│  (curl, app)  │◀────────│  :8080           │
└───────────────┘         │                  │
                          │  chi router      │
                          │  ├─ middleware    │
                          │  ├─ handlers     │
                          │  ├─ services     │
                          │  └─ repository   │
                          └────────┬─────────┘
                                   │ host=db (Docker DNS)
                          ┌────────▼─────────┐
                          │  PostgreSQL 15   │
                          │  :5432           │
                          └──────────────────┘
```

Both services run in Docker Compose on a shared bridge network. The backend resolves the database via Docker's internal DNS using the service name `db` — no hardcoded IPs or `localhost` references.

## Getting Started

### Prerequisites

- [Docker](https://docs.docker.com/get-docker/) (20.10+)
- [Docker Compose](https://docs.docker.com/compose/install/) (v2+)

### Setup & Run

```bash
# Clean any previous state
docker compose down -v --remove-orphans

# Build and start
docker compose up --build
```

The backend waits for PostgreSQL to pass its health check before starting. On first boot, all migrations run automatically and seed data is created.

### Verify

```bash
curl http://localhost:8080/health
```

**Expected response:**

```json
{
  "status": "ok",
  "db": "connected"
}
```

**Startup logs should show:**

```
[BOOT] Starting server...
[BOOT] Connecting to database...
[BOOT] Connected to database
[BOOT] Running migrations...
[BOOT] Migrations complete
[BOOT] Server listening on :8080
```

## API Endpoints

All responses use a consistent JSON structure:

```json
{
  "data": { },
  "error": null,
  "message": "description"
}
```

Error responses:

```json
{
  "error": "error description"
}
```

---

### Health

| Method | Endpoint  | Auth | Description             |
| ------ | --------- | ---- | ----------------------- |
| GET    | `/health` | No   | Service + DB status     |

---

### Authentication (Public)

| Method | Endpoint                 | Auth | Description                  |
| ------ | ------------------------ | ---- | ---------------------------- |
| POST   | `/auth/signup`           | No   | Register a new user          |
| POST   | `/auth/login`            | No   | Authenticate and get JWT     |
| GET    | `/auth/verify`           | No   | Verify email via token       |
| POST   | `/auth/forgot-password`  | No   | Request password reset email |
| POST   | `/auth/reset-password`   | No   | Reset password with token    |

### Authentication (Protected)

| Method | Endpoint    | Auth   | Description              |
| ------ | ----------- | ------ | ------------------------ |
| GET    | `/auth/me`  | Bearer | Get current user profile |

---

### Contacts (Protected — Bearer token required)

| Method | Endpoint                      | Description                    |
| ------ | ----------------------------- | ------------------------------ |
| POST   | `/contacts`                   | Create a new contact           |
| GET    | `/contacts`                   | List contacts (paginated)      |
| GET    | `/contacts/search?q=`         | Search contacts                |
| GET    | `/contacts/deleted`           | List soft-deleted contacts     |
| GET    | `/contacts/stats`             | Category breakdown stats       |
| GET    | `/contacts/{id}`              | Get a single contact           |
| PUT    | `/contacts/{id}`              | Update a contact               |
| DELETE | `/contacts/{id}`              | Soft-delete a contact          |
| PATCH  | `/contacts/{id}/restore`      | Restore a soft-deleted contact |
| DELETE | `/contacts/{id}/permanent`    | Permanently delete a contact   |

---

### Admin (Protected — Admin role required)

| Method | Endpoint                     | Description              |
| ------ | ---------------------------- | ------------------------ |
| GET    | `/admin/users`               | List all users           |
| PATCH  | `/admin/users/{id}/verify`   | Manually verify a user   |
| DELETE | `/admin/users/{id}`          | Delete a user            |

---

### Example Requests

**Signup:**

```bash
curl -X POST http://localhost:8080/auth/signup \
  -H "Content-Type: application/json" \
  -d '{"name": "Jane Doe", "email": "jane@example.com", "password": "securepass123"}'
```

```json
{
  "data": { "id": 1, "email": "jane@example.com" },
  "error": null,
  "message": "user created. Please check your email to verify your account"
}
```

**Login:**

```bash
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "jane@example.com", "password": "securepass123"}'
```

```json
{
  "data": { "token": "eyJhbGciOiJIUzI1NiIs..." },
  "error": null,
  "message": "login successful"
}
```

**Create contact:**

```bash
curl -X POST http://localhost:8080/contacts \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{"name": "John Smith", "phone": "+1234567890", "category_id": 1}'
```

**List contacts (paginated):**

```bash
curl "http://localhost:8080/contacts?page=1&limit=10&category=Work" \
  -H "Authorization: Bearer <token>"
```

## Environment Variables

| Variable           | Required | Default            | Description                                      |
| ------------------ | -------- | ------------------ | ------------------------------------------------ |
| `DATABASE_URL`     | Yes      | —                  | PostgreSQL connection string                     |
| `JWT_SECRET`       | Yes      | —                  | Secret key for signing JWT tokens                |
| `PORT`             | No       | `8080`             | Server listen port                               |
| `RESEND_API_KEY`   | No       | —                  | Resend API key for transactional emails           |
| `BASE_URL`         | No       | `http://localhost:3000` | Base URL for email verification links        |
| `DEBUG_EMAIL`      | No       | `false`            | Print emails to console instead of sending       |
| `FORCE_EMAIL`      | No       | `false`            | Override recipient for testing                   |
| `CLEANUP_INTERVAL` | No       | `24h`              | Interval for the background cleanup worker       |

Create a `.env` file in the `backend/` directory for local development. Docker Compose also sets `DATABASE_URL` explicitly to use Docker DNS (`host=db`).

## Database & Migrations

### Migration System

Migrations live in `backend/migrations/` as numbered `.sql` files and execute **automatically on every startup** in alphabetical order.

```
migrations/
├── 000_init.sql               # Base tables (categories, contacts)
├── 001_auth.sql               # Users table, admin seed, user_id FK
├── 002_verification.sql       # Email verification & password reset columns
├── 003_admin_role.sql         # Role column for RBAC
└── 004_phone_unique_per_user.sql  # Per-user phone uniqueness
```

### Idempotency Guarantees

Every migration is safe to re-run:

- Tables use `CREATE TABLE IF NOT EXISTS`
- Columns use `ADD COLUMN IF NOT EXISTS`
- Indexes use `CREATE INDEX IF NOT EXISTS`
- Seed data uses `INSERT ... ON CONFLICT DO NOTHING`
- Constraint changes check current state before altering

Running `docker compose up` multiple times will **never** produce duplicate data or constraint errors.

## Error Handling

- **All endpoints return JSON** — no empty responses, no plain text errors
- **Consistent envelope**: success responses use `{ data, error, message }`, error responses use `{ error }`
- **Proper HTTP status codes**: 400 for validation, 401/403 for auth, 404 for not found, 409 for conflicts, 429 for rate limits, 500 for server errors
- **No sensitive data leakage**: password reset always returns a generic message regardless of whether the email exists
- **Database failures** on `/health` return HTTP 500 with `{"status": "error", "db": "down"}`

## Project Structure

```
backend/
├── cmd/
│   └── api/
│       └── main.go              # Entry point, routing, wiring
├── internal/
│   ├── config/                  # Environment configuration loader
│   ├── database/                # DB connection, retry logic, migration runner
│   ├── handlers/                # HTTP handlers (auth, contacts, admin, health)
│   ├── middleware/               # JWT auth middleware, admin-only guard
│   ├── models/                  # Data structures and request/response types
│   ├── ratelimit/               # In-memory rate limiter
│   ├── repository/              # Database queries (SQL layer)
│   ├── services/                # Business logic
│   ├── utils/                   # JWT, hashing, mailer, response helpers
│   └── worker/                  # Background cleanup goroutine
├── migrations/                  # Idempotent SQL migration files
├── Dockerfile                   # Multi-stage build (golang → alpine)
├── .env                         # Environment variables (not committed)
├── go.mod
└── go.sum
```

## Future Improvements

- [ ] Structured logging (slog / zerolog) with request IDs
- [ ] Graceful shutdown with `os.Signal` handling
- [ ] Request validation middleware (payload size, schema)
- [ ] Prometheus metrics and `/metrics` endpoint
- [ ] Database connection pooling tuning
- [ ] API versioning (`/v1/...`)
- [ ] Pagination via cursor instead of offset
- [ ] Integration test suite with test containers
- [ ] CI/CD pipeline (GitHub Actions)
- [ ] OpenAPI / Swagger documentation

## License

This project is for educational and portfolio purposes.