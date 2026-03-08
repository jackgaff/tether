# Nova Echoes

Hackathon starter for a voice-first support app aimed at older adults who live
alone and benefit from gentle, recurring check-ins.

The repo is set up to help a small team move quickly without blurring concerns:

- Go API with centralized config loading and small module boundaries
- Bun-managed React + Vite frontend with typed API contracts
- Docker Compose stack for Postgres, API, and web
- One shared root `.env.example`, plus optional `.env.local` overrides
- Repo-level verification command for local work and CI
- OpenAPI spec served by the API at `/openapi.yaml`

## Prerequisites

For local development without Docker:

- Bun `1.3.10`
- Go `1.24.5`

For the containerized workflow:

- Docker Desktop with Docker Compose
- `make` if you want the convenience targets

## Repo Layout

```text
.
├── apps
│   ├── api
│   │   ├── cmd/server
│   │   ├── docs/openapi.yaml
│   │   └── internal
│   │       ├── app
│   │       ├── config
│   │       ├── httpserver
│   │       └── modules
│   └── web
│       └── src
├── .env.example
├── .github/workflows/ci.yml
├── Makefile
└── package.json
```

## First Setup

1. Bootstrap the repo:

   ```bash
   make init
   ```

   That creates `.env` from `.env.example` if needed, then installs Bun
   workspace dependencies.

2. Run the verification suite before you branch off:

   ```bash
   make check
   ```

The committed source of truth is `.env.example`. `.env` and `.env.local` are
local-only. Shell environment variables still win over file values, and
`.env.local` overrides `.env`.

## Local Toolchain Workflow

Use this when you want fast iteration with native processes on your machine.

1. Start Postgres if you need a local database.
   The repo defaults to host port `5433` so it does not collide with an
   existing local Postgres on `5432`.

2. Start the API:

   ```bash
   bun run dev:api
   ```

3. In another terminal, start the frontend:

   ```bash
   bun run dev:web
   ```

4. Open `http://localhost:5173`.

## Docker Workflow

Use this when you want the full stack with one command.

```bash
make up
```

That starts:

- `db` on host port `5433`
- `api` on `http://localhost:8080`
- `web` on `http://localhost:5173`

Useful commands:

```bash
make init
make status
make rebuild
make logs
make api-logs
make web-logs
make db-logs
make down
make db-reset
make clean
```

Notes:

- `make up` starts quickly with the current images. Use `make rebuild` after
  Dockerfile or dependency changes.
- The web container only receives `VITE_*` variables. Backend-only values stay
  scoped to the API service.
- The Compose web image installs dependencies from `bun.lock` during build, so
  startup does not depend on a fresh runtime install.
- `make db-reset` recreates the Postgres volume and restarts the stack.

## Environment Variables

Root env files are shared intentionally, but not every value is consumed by
every service.

Backend/runtime variables:

- `APP_NAME`
- `APP_ENV`
- `API_PORT`
- `FRONTEND_ORIGIN`
- `DATABASE_URL`
- `AUTH_MODE`
- `INTERNAL_API_KEY`
- `AWS_REGION`
- `BEDROCK_REGION`
- `NOVA_VOICE_MODEL_ID`
- `NOVA_ANALYSIS_MODEL_ID`

Frontend variables:

- `VITE_API_BASE_URL`
- `VITE_APP_NAME`

Guidance:

- Keep browser-facing routes public by default.
- Treat `INTERNAL_API_KEY` as server-to-server only. Do not expose it to the
  browser.
- When wiring Amazon Nova or Bedrock, use normal AWS credentials or IAM-based
  auth rather than inventing your own credential flow.

## Verification

Run the same checks locally that CI runs:

```bash
bun run check
```

That currently does:

- `go test -race ./...`
- `go vet ./...`
- frontend typecheck
- frontend production build

CI lives at `.github/workflows/ci.yml` and runs on pushes to `main` plus pull
requests.

## API Surface

Implemented routes:

- `GET /`
- `GET /openapi.yaml`
- `GET /health`
- `GET /api/v1/check-ins`
- `POST /api/v1/check-ins`

OpenAPI source: `apps/api/docs/openapi.yaml`

Example requests:

```bash
curl http://localhost:8080/health
```

```bash
curl -X POST http://localhost:8080/api/v1/check-ins \
  -H "Content-Type: application/json" \
  -d '{
    "patientId": "patient-001",
    "summary": "Caller remembered breakfast and tomorrow'\''s appointment.",
    "status": "completed",
    "agent": "call-agent",
    "reminder": "Keep the appointment card by the front door."
  }'
```

If you set `AUTH_MODE=api-key`, send:

```bash
-H "X-API-Key: $INTERNAL_API_KEY"
```

## Suggested Next Slices

- Replace the in-memory check-in store with Postgres persistence
- Add scheduler adapters for recurring outbound calls
- Introduce Bedrock/Nova clients behind injected interfaces
- Split agent flows into call, analysis, and safety services
- Add caregiver-facing summaries and escalation thresholds
