# FinancialControl (Fluxo)

Collaborative financial control platform with a clear separation of concerns.

```text
frontend/                     Web UI (React + Vite)
        |
        v
cmd/api                       Go HTTP server entrypoint
        |
        +--> internal/server      Route registration
        +--> internal/handlers    HTTP handlers
        +--> internal/repository  SQLite access
        +--> internal/analysis    Python HTTP client
        |
        v
SQLite file (./data/fluxo.db)  <--- migrations/ (cmd/migrate)
        |
        +--> services/python-analysis
```

## Why this structure?

| Path | Responsibility |
|------|----------------|
| `cmd/api` | Process entrypoint only (wiring) |
| `cmd/migrate` | Apply/rollback SQL migrations |
| `internal/config` | Environment configuration |
| `internal/database` | SQLite connection |
| `internal/server` | Routes + middleware composition |
| `internal/handlers` | HTTP request/response logic |
| `internal/repository` | SQL queries |
| `internal/models` | Domain types / DTOs |
| `migrations/` | Versioned schema changes |
| `frontend/` | Visual app to manage expenses |
| `services/python-analysis/` | Monthly financial analysis |

## Why SQLite?

- No Docker / no server to install
- One file on disk (`./data/fluxo.db`)
- Perfect for learning and local portfolio demos
- Still uses real SQL + migrations (same habits as PostgreSQL later)

## Prerequisites

- Go 1.22+
- Node.js (frontend)
- Python 3.11+ (analysis service)

## 1. Configure environment (optional)

```bash
cp .env.example .env
# defaults already work: SQLITE_PATH=./data/fluxo.db
```

## 2. Run migrations

```bash
go run ./cmd/migrate -direction=up
```

Rollback example:

```bash
go run ./cmd/migrate -direction=down -steps=1
```

## 3. Start Go API

```bash
go run ./cmd/api
```

## 4. Start Python analysis

```bash
cd services/python-analysis
python -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
uvicorn main:app --reload --port 8000
```

## 5. Start web UI

```bash
cd frontend
npm install
npm run dev
```

Open the Vite URL (usually `http://localhost:5173`).

## API routes

```text
GET    /health
GET    /categories
POST   /categories
GET    /categories/{id}
PUT    /categories/{id}
DELETE /categories/{id}

GET    /transactions
POST   /transactions
GET    /transactions/{id}
PUT    /transactions/{id}
DELETE /transactions/{id}

GET    /analysis/monthly?year=2026&month=8
```

## Tests

```bash
go test ./...
```

## Libraries used (and why)

- `database/sql` + `modernc.org/sqlite`: standard SQL API with a pure-Go SQLite driver (no CGO)
- `golang-migrate`: versioned migrations (`up` / `down`)

We do **not** use an HTTP framework yet: `net/http` is enough and better for learning.

## Not included yet

- Authentication / JWT
- PostgreSQL (can replace SQLite later if needed)
- Redis / Kafka / Kubernetes
