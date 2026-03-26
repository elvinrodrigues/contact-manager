<div align="center">

# ContactHub

**A full-stack contact management application built with Go and Next.js**

[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?style=flat&logo=go&logoColor=white)](https://go.dev/)
[![Next.js](https://img.shields.io/badge/Next.js-16-black?style=flat&logo=next.js&logoColor=white)](https://nextjs.org/)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.7-3178C6?style=flat&logo=typescript&logoColor=white)](https://www.typescriptlang.org/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-336791?style=flat&logo=postgresql&logoColor=white)](https://www.postgresql.org/)
[![Tailwind CSS](https://img.shields.io/badge/Tailwind-v4-38B2AC?style=flat&logo=tailwind-css&logoColor=white)](https://tailwindcss.com/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg?style=flat)](LICENSE)

[Features](#features) · [Tech Stack](#tech-stack) · [Getting Started](#getting-started) · [API Reference](#api-reference) · [Project Structure](#project-structure)

</div>

---

## Overview

ContactHub is a full-stack contact manager with a Go REST API backend and a modern Next.js frontend. It supports creating, searching, updating, and soft-deleting contacts, with a built-in duplicate detection system based on phone number normalization.

The frontend proxies all API calls through Next.js rewrites, so no CORS configuration is needed during development beyond a single environment variable.

---

## Features

- **Full CRUD** — Create, read, update, and delete contacts
- **Duplicate detection** — Automatically detects contacts with the same normalized phone number before saving
- **Phone normalization** — Strips country codes (`+91`, `91`, `0`) and validates 10-digit numbers
- **Smart search** — Case-insensitive search across name and phone number (`ILIKE`)
- **Soft delete & restore** — Deleted contacts are archived, not permanently removed, and can be restored at any time
- **Category labels** — Contacts are tagged as Default, Family, Friends, or Work
- **Pagination** — Server-side pagination on list endpoints
- **Dark mode** — System-aware theme with manual toggle, persisted via `next-themes`
- **Toast notifications** — Instant feedback on every action (create, delete, restore, error)
- **Responsive UI** — Works cleanly on mobile and desktop

---

## Tech Stack

### Backend

| Layer | Technology |
|---|---|
| Language | Go 1.25 |
| Router | [chi v5](https://github.com/go-chi/chi) |
| CORS | [go-chi/cors](https://github.com/go-chi/cors) |
| Database | PostgreSQL 16 |
| DB Driver | [pgx v5](https://github.com/jackc/pgx) + [lib/pq](https://github.com/lib/pq) |
| Architecture | Layered — Handler → Service → Repository |

### Frontend

| Layer | Technology |
|---|---|
| Framework | Next.js 16 (App Router) |
| Language | TypeScript 5.7 |
| Styling | Tailwind CSS v4 |
| UI Primitives | Radix UI |
| Forms | React Hook Form |
| Toasts | Sonner |
| Icons | Lucide React |
| Theme | next-themes |

---

## Project Structure

```
contact-manager/
├── backend/
│   ├── cmd/
│   │   └── api/
│   │       └── main.go          # Entry point — wires router, middleware, handlers
│   ├── internal/
│   │   ├── database/
│   │   │   └── db.go            # PostgreSQL connection
│   │   ├── handlers/
│   │   │   └── contact_handler.go  # HTTP handlers (request parsing, response encoding)
│   │   ├── models/
│   │   │   └── models.go        # Contact, CreateContactResult, ListContactsResult
│   │   ├── repository/
│   │   │   ├── repository.go    # ContactRepository struct + constructor
│   │   │   └── contact_repository.go  # All SQL queries
│   │   ├── services/
│   │   │   ├── service.go       # ContactService struct + constructor
│   │   │   └── contact_service.go     # Business logic (duplicate check, normalize, paginate)
│   │   └── utils/
│   │       └── phone.go         # Phone number normalization + validation
│   ├── migrations/              # SQL migration files (schema)
│   ├── go.mod
│   └── go.sum
│
└── frontend/
    ├── app/
    │   ├── layout.tsx           # Root layout — ThemeProvider, Toaster, fonts
    │   ├── page.tsx             # Dashboard — tabs for Active / Deleted contacts
    │   └── globals.css          # Design tokens (CSS variables), Tailwind base
    ├── components/
    │   ├── navbar.tsx           # Sticky header with logo and dark mode toggle
    │   ├── contact-form.tsx     # Create contact dialog with validation
    │   ├── contacts-table.tsx   # Active contacts table with search and delete
    │   ├── deleted-contacts-table.tsx  # Archived contacts table with restore
    │   ├── duplicate-dialog.tsx # Warning shown when a duplicate phone is detected
    │   ├── confirm-dialog.tsx   # Reusable confirmation modal (delete / restore)
    │   ├── contact-ui.tsx       # Shared CategoryBadge and ContactAvatar components
    │   └── ui/                  # Radix-based primitives (Button, Dialog, Table, etc.)
    ├── lib/
    │   ├── api-service.ts       # All fetch calls to the backend
    │   └── utils.ts             # cn() helper (clsx + tailwind-merge)
    ├── hooks/
    ├── styles/
    ├── next.config.js           # API proxy rewrites (/api/* → localhost:8080/*)
    └── package.json
```

---

## Getting Started

### Prerequisites

| Tool | Minimum Version |
|---|---|
| Go | 1.25 |
| Node.js | 18 |
| pnpm | 8 |
| PostgreSQL | 14 |

> **pnpm** is the package manager used by the frontend. Install it with `npm install -g pnpm` if you don't have it.

---

### 1. Database Setup

Connect to PostgreSQL and run the following:

```sql
-- Create the dedicated user
CREATE USER contacts_app WITH PASSWORD 'your_password';

-- Create the database
CREATE DATABASE contacts_manager OWNER contacts_app;

-- Connect to the new database
\c contacts_manager

-- Create the contacts table
CREATE TABLE contacts (
    id          SERIAL PRIMARY KEY,
    name        TEXT        NOT NULL,
    phone       TEXT        NOT NULL,
    email       TEXT,
    category_id INTEGER     NOT NULL DEFAULT 1,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ
);

-- Optional: index for faster soft-delete filtering and phone lookup
CREATE INDEX idx_contacts_deleted_at ON contacts (deleted_at);
CREATE INDEX idx_contacts_phone      ON contacts (phone);
```

---

### 2. Backend Setup

```bash
cd backend
```

Update the connection string in `internal/database/db.go` to match your PostgreSQL credentials:

```go
connStr := "host=localhost user=contacts_app password=your_password dbname=contacts_manager sslmode=disable"
```

Install dependencies and run the server:

```bash
go mod tidy
go run ./cmd/api
```

The API will start on **`http://localhost:8080`**.

```
2025/xx/xx xx:xx:xx Connected to PostgreSQL
2025/xx/xx xx:xx:xx Server running on :8080
```

---

### 3. Frontend Setup

```bash
cd frontend
```

Install dependencies:

```bash
pnpm install
```

Start the development server:

```bash
pnpm dev
```

The app will be available at **`http://localhost:3000`**.

All requests to `/api/*` are automatically proxied to `http://localhost:8080/*` via Next.js rewrites — no additional configuration needed.

---

## API Reference

All endpoints are prefixed with `/contacts`. The frontend reaches them through `/api/contacts`.

### Contacts

| Method | Path | Description | Request Body | Response |
|---|---|---|---|---|
| `POST` | `/contacts` | Create a contact. Returns duplicate info if phone already exists. | `{ name, phone, email?, category_id? }` | `{ status, contact?, duplicates? }` |
| `GET` | `/contacts` | List active contacts (paginated, ordered by name). | — | `{ contacts, page, limit, total }` |
| `GET` | `/contacts/search?q=` | Search active contacts by name or phone (`ILIKE`). | — | `{ contacts }` |
| `GET` | `/contacts/{id}` | Get a single active contact by ID. | — | `Contact` |
| `PUT` | `/contacts/{id}` | Update a contact's name, email, or category. | `{ name, email?, category_id? }` | `{ message }` |
| `DELETE` | `/contacts/{id}` | Soft-delete a contact (sets `deleted_at`). | — | `204 No Content` |
| `GET` | `/contacts/deleted` | List soft-deleted contacts (paginated). | — | `{ contacts, page, limit, total }` |
| `PATCH` | `/contacts/{id}/restore` | Restore a soft-deleted contact (clears `deleted_at`). | — | `{ message }` |

### Pagination

`GET /contacts` and `GET /contacts/deleted` accept optional query parameters:

| Parameter | Default | Description |
|---|---|---|
| `page` | `1` | Page number (1-indexed) |
| `limit` | `10` | Results per page |

### Contact Schema

```json
{
  "id":          1,
  "name":        "Jane Doe",
  "phone":       "9876543210",
  "email":       "jane@example.com",
  "category_id": 2,
  "created_at":  "2025-01-01T10:00:00Z",
  "updated_at":  "2025-01-01T10:00:00Z",
  "deleted_at":  null
}
```

### Create Contact Response

```json
{
  "status": "created",
  "contact": { ... }
}
```

If a contact with the same phone number already exists:

```json
{
  "status": "duplicate",
  "duplicates": [ { ... } ]
}
```

### Phone Normalization

The backend normalizes phone numbers before duplicate checking and storage:

| Input | Stored As |
|---|---|
| `+91 98765 43210` | `9876543210` |
| `91-9876543210` | `9876543210` |
| `09876543210` | `9876543210` |
| `9876543210` | `9876543210` |

Numbers that don't resolve to exactly 10 digits are rejected with `400 Bad Request`.

### Categories

| ID | Label |
|---|---|
| `1` | Default |
| `2` | Family |
| `3` | Friends |
| `4` | Work |

---

## Screenshots

> Screenshots will be added in a future update.

| View | Description |
|---|---|
| Dashboard | Active contacts table with search, group-by-category toggle, and per-row delete |
| Deleted | Archived contacts with restore action and deleted-on date |
| Create dialog | New contact form with phone/email validation and category picker |
| Duplicate warning | Warning dialog when a matching phone number is already in the system |
| Dark mode | Full dark mode support across all views |

---

## Future Improvements

- [ ] **Authentication** — JWT-based login so the app can be deployed multi-user
- [ ] **Edit contact** — In-place edit dialog for name, email, and category (the `PUT` endpoint exists; the frontend form is not yet wired up)
- [ ] **Import / Export** — CSV import for bulk contact upload and export for backups
- [ ] **Database migrations** — Introduce a migration tool (e.g. `golang-migrate`) to manage schema changes properly
- [ ] **Environment variables** — Move the DB connection string to `.env` / config file instead of hardcoded string
- [ ] **Full-text search** — Replace `ILIKE` with PostgreSQL `tsvector` / `tsquery` for better search performance at scale
- [ ] **Tests** — Unit tests for the service layer, integration tests for the HTTP handlers
- [ ] **Docker Compose** — Single-command local setup for the API, database, and frontend

---

## License

This project is licensed under the [MIT License](LICENSE).

---

<div align="center">

Built with Go + Next.js · [Report a bug](../../issues) · [Request a feature](../../issues)

</div>
