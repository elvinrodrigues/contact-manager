# ContactHub — Backend API

A production-grade REST API for managing personal contacts, built with Go and PostgreSQL. Features JWT-based authentication, role-based access control, email verification, soft-delete with automated cleanup, and a fully Dockerized deployment.

## Overview

ContactHub is a backend service that provides secure, multi-user contact management through a JSON REST API. Each user manages their own isolated set of contacts with full CRUD operations, search, soft-delete/restore workflows, and automatic purging of expired records.

**Key capabilities:**

- JWT authentication with email verification and password reset
- Per-user contact isolation with soft-delete lifecycle
- Role-based access control (user / admin)
- Background worker for automated cleanup of expired contacts
- Tracked database migrations that apply exactly once, each in a transaction
- Production health checks with database connectivity verification
- Rate limiting on signup, login and password reset
- Graceful shutdown, HTTP timeouts and request body limits

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
- Rate limiting on signup, login and forgot-password (per-IP and per-account)
- Timing-attack mitigation on account enumeration
- Token revocation on password reset (JWTs carry a token version)
- Admin role granted per-deployment via `ADMIN_EMAIL`, never seeded in a migration

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
- Tuned connection pool (25 open / 25 idle, 5m lifetime)
- Consistent JSON response envelope on every endpoint, success or failure
- Configurable CORS origins and trusted-proxy allowlist

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

The backend waits for PostgreSQL to pass its health check before starting. On first boot the migrations run automatically and seed the category lookup table.

**No user account is created by the migrations.** Sign up through the API, then
name that address in `ADMIN_EMAIL` to grant it the admin role at the next start.

### Verify

```bash
curl http://localhost:8080/health
```

**Expected response:**

```json
{
  "data": { "status": "ok", "db": "connected" },
  "error": null,
  "message": "Service healthy"
}
```

**Startup logs should show:**

```
[BOOT] Starting server
[BOOT] Connected to database (pool: 25 open / 25 idle)
[BOOT] Running migrations
  applied 000_init.sql
  ...
[BOOT] Applied 6 new migration(s)
[BOOT] Listening on :8080
[Worker] Cleanup job started; running every 24h0m0s (retention 30 days)
```

On subsequent starts the migration step reports `Schema up to date` and applies
nothing.

## API Endpoints

All responses use a consistent JSON structure:

```json
{
  "data": { },
  "error": null,
  "message": "description"
}
```

Errors use the same envelope, with a stable machine-readable `code`:

```json
{
  "data": null,
  "error": { "code": "validation_failed", "message": "name is required" },
  "message": "name is required"
}
```

Codes: `bad_request`, `validation_failed`, `unauthorized`, `forbidden`,
`not_found`, `conflict`, `payload_too_large`, `rate_limited`, `internal_error`.

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
| GET    | `/admin/users`               | List users (paginated)   |
| PATCH  | `/admin/users/{id}/verify`   | Manually verify a user   |
| DELETE | `/admin/users/{id}`          | Delete a user            |

---

### Example Requests

**Signup:**

```bash
curl -X POST http://localhost:8080/auth/signup \
  -H "Content-Type: application/json" \
  -d '{"name": "Jane Doe", "email": "jane@example.com", "password": "a-long-enough-passphrase"}'
```

```json
{
  "data": { "id": 1, "email": "jane@example.com" },
  "error": null,
  "message": "Account created. Check your email to verify your account"
}
```

Passwords must be 12–72 characters. The account cannot log in until the
verification link is followed.

**Login:**

```bash
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "jane@example.com", "password": "a-long-enough-passphrase"}'
```

```json
{
  "data": { "token": "eyJhbGciOiJIUzI1NiIs..." },
  "error": null,
  "message": "Login successful"
}
```

**Create contact:**

```bash
curl -X POST http://localhost:8080/contacts \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{"name": "John Smith", "phone": "+919876543210", "category_id": 1}'
```

Creating a contact whose phone number already exists returns **409** with the
conflicting records attached, so a client can offer to merge them.

**List contacts (paginated):**

```bash
# category takes a numeric category id, or "all"
curl "http://localhost:8080/contacts?page=1&limit=10&category=4" \
  -H "Authorization: Bearer <token>"
```

`limit` is capped at 100. An empty page returns `"contacts": []`, never `null`.

## Environment Variables

| Variable               | Required | Default                 | Description                                                   |
| ---------------------- | -------- | ----------------------- | ------------------------------------------------------------- |
| `JWT_SECRET`           | **Yes**  | —                       | Signing key for session tokens; the server refuses to start without it |
| `DATABASE_URL`         | No       | built from `POSTGRES_*` | Full connection string; takes priority when set               |
| `POSTGRES_USER`        | No       | `contacts_app`          | Used to build the DSN                                          |
| `POSTGRES_PASSWORD`    | No       | —                       | Used to build the DSN                                          |
| `POSTGRES_DB`          | No       | `contacts_manager`      | Used to build the DSN                                          |
| `DB_HOST`              | No       | `localhost`             | `db` under Docker Compose                                      |
| `DB_PORT`              | No       | `5432`                  | Database port                                                  |
| `DB_SSLMODE`           | No       | `disable`               | Use `require` for any database reachable off-host              |
| `PORT`                 | No       | `8080`                  | Server listen port                                             |
| `ADMIN_EMAIL`          | No       | —                       | Grants the admin role to this account at startup               |
| `CORS_ALLOWED_ORIGINS` | No       | `http://localhost:3000` | Comma-separated browser origins; never set to `*`              |
| `TRUSTED_PROXY_CIDRS`  | No       | —                       | Whose `X-Forwarded-For` to believe; empty means ignore it      |
| `MAX_BODY_BYTES`       | No       | `1048576`               | Largest accepted request body                                  |
| `RESEND_API_KEY`       | No       | —                       | Resend API key for transactional email                         |
| `MAIL_FROM`            | No       | `ContactHub <onboarding@resend.dev>` | From address on outbound mail                     |
| `BASE_URL`             | No       | `http://localhost:3000` | Public frontend URL used to build email links                  |
| `DEBUG_EMAIL`          | No       | `false`                 | Log that mail would be sent instead of sending it              |
| `FORCE_EMAIL`          | No       | `false`                 | Redirect all mail to one inbox (requires `FORCE_EMAIL_TO`)     |
| `FORCE_EMAIL_TO`       | No       | —                       | Recipient for `FORCE_EMAIL`; no default, by design             |
| `CLEANUP_INTERVAL`     | No       | `24h`                   | How often the purge worker runs                                |
| `MIGRATIONS_DIR`       | No       | `migrations`            | Where to look for migration files                              |

Copy `.env.backend.example` to `.env.backend` and fill it in. Nothing outside
`internal/config` reads the environment directly.

### A note on secrets

`ADMIN_EMAIL` exists because administrators must be a property of the
deployment. An earlier revision promoted a hardcoded address in
`003_admin_role.sql` and seeded two accounts with passwords written in this
repository — anyone who read it held credentials on every instance.
`005_revoke_seeded_credentials.sql` revokes those accounts if your database
already ran the old migrations; recover access with the password-reset flow.

## Deployment

Both services are defined in the repo so the configuration is reviewed like
code: `render.yaml` for the API, `frontend/vercel.json` for the web app. That
matters — the frontend was swapped from Next.js to Vite in April while the
Vercel project stayed configured for Next.js, and nothing caught it until the
next push four months later.

### Backend (Render)

New -> Blueprint -> point at this repo. It creates the Postgres instance and the
Docker web service, wires `DATABASE_URL`, and generates `JWT_SECRET`.

Then set the four dashboard-only values (they are `sync: false` so they never
live in git):

| Variable | Value |
| -------- | ----- |
| `ADMIN_EMAIL` | the account to grant admin on next boot; it must have signed up already |
| `BASE_URL` | the Vercel URL, e.g. `https://contact-manager.vercel.app` |
| `CORS_ALLOWED_ORIGINS` | the same Vercel URL |
| `RESEND_API_KEY` | your Resend key, if you want real email |

Render's free Postgres is **deleted after 30 days of inactivity**. When that
happens the API exits at boot with `no such host` for the database hostname —
recreate the database and re-link `DATABASE_URL`.

### Frontend (Vercel)

Set **Root Directory** to `frontend`. Everything else comes from
`frontend/vercel.json`: framework preset, build command, output directory, and
the SPA rewrite.

Set one environment variable:

| Variable | Value |
| -------- | ----- |
| `VITE_API_BASE_URL` | the Render URL, e.g. `https://contacthub-api.onrender.com` (no trailing slash) |

Without it the bundle falls back to `/api`, which only exists behind the Vite
dev proxy — every request would 404 against the static host.

The catch-all rewrite in `vercel.json` is not optional. The app uses
`BrowserRouter`, and the emailed verification link is a direct navigation to
`/verify?token=...`; without the rewrite that path is served by the CDN, finds
no file, and returns 404 before React ever loads.

### First admin

No account is seeded. Sign up through the deployed frontend, set `ADMIN_EMAIL`
to that address, and redeploy — the role is granted at startup.

## Database & Migrations

### Migration System

Migrations live in `backend/migrations/` as numbered `.sql` files, applied in
filename order at startup.

```
migrations/
├── 000_init.sql                   # Base tables (categories, contacts)
├── 001_auth.sql                   # Users table, user_id FK and backfill
├── 002_verification.sql           # Email verification & password reset columns
├── 003_admin_role.sql             # Role column for RBAC
├── 004_phone_unique_per_user.sql  # Per-user phone uniqueness (partial index)
├── 005_revoke_seeded_credentials.sql  # Neutralises the old seeded accounts
└── 006_schema_hardening.sql       # Constraints, indexes, cascades, triggers
```

### Applied exactly once

Each file is recorded in a `schema_migrations` table once it succeeds and is
skipped on every later start, and each runs inside its own transaction so a
failure rolls back rather than leaving the schema half-changed.

This matters beyond tidiness. Migrations previously re-ran on every boot, and
`002_verification.sql` carried a one-time backfill —
`UPDATE users SET is_verified = true WHERE is_verified = false` — which meant
every restart silently verified every pending signup. The statement is now
scoped to the run that introduces the column, and the ledger stops the file
being re-applied at all.

### Schema guarantees

- `contacts.user_id` → `users(id)` **ON DELETE CASCADE**: deleting a user
  removes their contacts in one statement, leaving no orphans.
- `contacts.category_id` → `categories(id)` **ON DELETE RESTRICT**: a category
  in use cannot disappear underneath the rows referencing it.
- A partial unique index on `(user_id, phone) WHERE deleted_at IS NULL` is what
  actually prevents duplicates. The application's pre-check exists to produce a
  useful 409 body; the index is the guarantee under concurrency.
- `CHECK` constraints reject blank names and empty-string emails, so "no email"
  is always `NULL` and never `''`.
- A `BEFORE UPDATE` trigger maintains `updated_at`, so every write path advances
  it rather than only the one statement that used to set it explicitly.

### Indexes

Every contact index leads with `user_id`, because that is the equality predicate
on every query. Trigram (`pg_trgm`) GIN indexes back the `ILIKE '%term%'` search,
which no B-tree can serve.

## Error Handling

- **One envelope everywhere.** Success and failure both return
  `{ data, error, message }`; `error` is `null` or `{ code, message }`.
- **Status codes carry meaning**: 400 validation, 401 unauthenticated,
  403 forbidden, 404 not found, 409 conflict, 413 body too large,
  415 wrong content type, 429 rate limited, 500 server error.
- **Nothing internal leaks.** Database and driver errors are translated into
  domain errors at the repository boundary; clients never see SQL text,
  constraint names or driver types.
- **Ownership failures read as 404, not 403.** Confirming that an id exists but
  belongs to someone else is itself a disclosure, so a contact you do not own is
  indistinguishable from one that does not exist.
- **Password reset never reveals whether an account exists** — generic response,
  evened-out timing.
- `/health` returns **503** with `{"status": "degraded", "db": "down"}` when the
  database is unreachable.

## Testing

```bash
go test ./...          # unit tests: domain, services, handlers, middleware, utils
go test -race ./...    # the same, under the race detector
go vet ./...
gofmt -l .
```

Repository and migration tests need a real PostgreSQL — the properties they
check (ownership scoping in the `WHERE` clause, the partial unique index,
cascade behaviour, `LIKE` escaping, migration idempotency) live in the database,
not in Go. They are behind a build tag:

```bash
TEST_DATABASE_URL="postgres://user:pass@localhost:5432/testdb?sslmode=disable" \
  go test -tags=integration ./internal/repository/...
```

Point it at a throwaway database: the suite drops and recreates the `public`
schema between tests.

## Project Structure

```
backend/
├── cmd/
│   └── api/
│       └── main.go              # Entry point: config, wiring, routing, shutdown
├── internal/
│   ├── config/                  # The only place that reads the environment
│   ├── database/                # Connection, pool tuning, migration runner
│   ├── handlers/                # HTTP layer: decoding, status codes, envelope
│   ├── middleware/              # Auth, admin guard, body-size limit
│   ├── models/                  # Domain types and their validation rules
│   ├── ratelimit/               # In-memory fixed-window limiter
│   ├── repository/              # SQL; the only package that imports lib/pq
│   ├── services/                # Business logic; owns the repository interfaces
│   ├── utils/                   # JWT, tokens, phone, client IP, mailer, responses
│   └── worker/                  # Background purge goroutine
├── migrations/                  # Tracked SQL migrations
├── Dockerfile                   # Multi-stage build (golang → alpine)
├── go.mod
└── go.sum
```

Dependencies point one way: `handlers → services → repository → database`.
Services declare the interfaces they need (`ContactRepo`, `UserRepo`, `Mailer`)
and depend on those rather than on concrete types, which is what makes the layer
testable without a database.

## Future Improvements

- [ ] Structured logging (`log/slog`) with request IDs on every line
- [ ] Prometheus metrics and a `/metrics` endpoint
- [ ] API versioning (`/v1/...`)
- [ ] Keyset pagination instead of offset, so pages cannot skip or repeat rows
      when contacts are inserted concurrently
- [ ] Shared rate-limit store (Redis) — the current limiter is process-local, so
      each replica enforces its own budget
- [ ] CI pipeline running the unit and integration suites plus `golangci-lint`
- [ ] OpenAPI / Swagger documentation

Done since the first revision: graceful shutdown, HTTP and body-size limits,
connection-pool tuning, context propagation, and an integration suite.

## License

This project is for educational and portfolio purposes.