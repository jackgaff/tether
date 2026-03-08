# Nova Echoes

Hackathon starter for a voice-first support app aimed at older adults who live
alone and benefit from gentle, recurring check-ins.

The repo is set up to help a small team move quickly without blurring concerns:

- Go API with centralized config loading and small module boundaries
- Bun-managed React + Vite frontend with typed API contracts
- Separate Bun-managed prompt lab app for voice-session testing
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
│   ├── test
│   │   └── src
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

3. In another terminal, start the main frontend:

   ```bash
   bun run dev:web
   ```

4. If you want the standalone prompt lab instead, start:

   ```bash
   bun run dev:test
   ```

5. Open `http://localhost:5173` for the main app or `http://localhost:5174` for the prompt lab.

## Docker Workflow

Use this when you want the full stack with one command.

```bash
make up
```

That starts:

- `db` on host port `5433`
- `api` on `http://localhost:8080`
- `web` on `http://localhost:5173`

If you also want the standalone prompt lab, start it separately:

```bash
make prompt-test
```

That serves the prompt lab on `http://localhost:5174`.

Useful commands:

```bash
make init
make status
make rebuild
make logs
make api-logs
make web-logs
make prompt-test
make prompt-test-logs
make db-logs
make down
make db-reset
make clean
```

Notes:

- `make up` starts quickly with the current images. Use `make rebuild` after
  Dockerfile or dependency changes.
- `make prompt-test` starts the standalone prompt lab on `http://localhost:5174`
  without changing the default `make up` stack.
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
- `ALLOWED_FRONTEND_ORIGINS`
- `VOICE_LAB_EXPORT_DIR`

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
- prompt lab typecheck
- prompt lab production build

CI lives at `.github/workflows/ci.yml` and runs on pushes to `main` plus pull
requests.

## API Surface

Implemented routes:

- `GET /`
- `GET /openapi.yaml`
- `GET /health`
- `GET /api/v1/voice/voices`
- `GET /api/v1/patients/{id}/preferences`
- `PUT /api/v1/patients/{id}/preferences`
- `POST /api/v1/voice/sessions`
- `GET /api/v1/voice/sessions/{id}/stream` (WebSocket upgrade)
- `GET /api/v1/check-ins`
- `POST /api/v1/check-ins`

OpenAPI source: `apps/api/docs/openapi.yaml`
Voice WebSocket contract: `apps/api/docs/voice-ws.md`

Voice transcript persistence:

- FINAL transcript turns are stored in Postgres table `voice_transcript_turns`
- voice session metadata is stored in `voice_sessions`
- usage events are stored in `voice_usage_events`
- prompt-lab sessions also export JSON and Markdown artifacts to
  `VOICE_LAB_EXPORT_DIR` which defaults to `apps/api/testdata/voice-lab`

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
