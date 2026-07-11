# Launchpad

A self-hosted, zero-downtime deployment platform built from scratch. Push code and Launchpad builds the new version in an idle slot, health-checks it privately, then flips nginx over with a graceful reload — your users never see a blink. Bad release? Roll back to the standby slot in one click, no rebuild.

<br/>

<p align="center">
  <img src="screenshots/02-dashboard.png" alt="Launchpad dashboard – blue/green slots with deploy history" width="100%"/>
</p>

---

## What it does

- **Deploys on git push** — a webhook triggers a full build → health check → swap cycle, no manual steps
- **Zero downtime** — builds happen in the idle slot while the live slot keeps serving; nginx only flips once the new slot is healthy
- **Blue/green slots** — two slots per service (blue, green); exactly one is live, the other is a warm standby
- **Private health check** — the new slot is verified over `localhost`, bypassing nginx, so users are never routed to an unhealthy container
- **One-click rollback** — flips nginx straight back to the standby slot; instant, no rebuild, because the previous version was never torn down
- **Never touches your data** — Launchpad manages only the app container; db/cache/queues persist untouched across every deploy
- **Crash-safe deploys** — a per-service lock with a dead-man's-switch heartbeat means a crashed build is detected and reset, never bricking the service
- **Live dashboard** — deploy history, the two-slot view, build durations, and rollback, all polling in real time during a build

---

## Screenshots

### Blue/green slots

<p align="center">
  <img src="screenshots/03-slots.png" alt="Launchpad – blue standby and green live slot cards" width="100%"/>
</p>

Two cards, one per slot. The live slot shows **LIVE** and carries the rollback button; the standby slot shows **STANDBY** and the deploy it's holding. Each card shows the commit SHA, message, when it landed, and how long the build took. Rolling back swaps the two.

### Dashboard — service detail

<p align="center">
  <img src="screenshots/02-dashboard.png" alt="Launchpad – full service dashboard" width="100%"/>
</p>

Service configuration (domain, repo, host, webhook) sits above the two slot cards, with the full deploy history below — every deploy with its SHA, status, slot, and time, expandable for detail.

### Zero-downtime proof

Hammer test — 1 request/second through nginx while triggering 3 live deploys:

```
828 requests · 0 failures
```

Raw output in [`screenshots/zero-downtime-run.txt`](screenshots/zero-downtime-run.txt).

---

## Architecture

```
  git push
     │  POST /webhooks/{serviceID}
     ▼
  ┌────────────────────┐         SSH          ┌─────────────────────────────────┐
  │  Launchpad Server  │ ───────────────────► │        Customer server          │
  │  (Go / HTTP)       │                      │                                 │
  │                    │                      │  nginx ──► active slot's port   │
  │  • webhook intake  │                      │   ├─ app:blue    (slot 1)       │
  │  • deploy engine   │                      │   ├─ app:green   (slot 2)       │
  │  • blue/green      │                      │   └─ db / cache / queues        │
  │    orchestration   │                      │      (persist, never swapped)   │
  └─────────┬──────────┘                      └─────────────────────────────────┘
            │ PostgreSQL
  ┌────────────────────┐
  │  services          │
  │  deploys           │
  │  deploy_locks      │
  └────────────────────┘
```

**Launchpad Server** — an HTTP mux plus a background deploy engine that:
1. Accepts a git webhook at `/webhooks/{serviceID}` and enqueues a deploy record
2. Picks up pending deploys, acquires a per-service lock, and runs the build over SSH on the customer server
3. Clones the repo, strips the app's host-port binding, starts supporting services, builds the app into the **idle** slot
4. Health-checks the new slot over `localhost`, then rewrites the service's nginx config to point at the new slot and reloads
5. Marks the deploy active, updates the service's active slot, releases the lock

**Deploy engine** (`internal/agent`) — four cooperating pieces:
- **Scheduler** — polls for pending deploys and dispatches one worker per service (never two for the same service at once)
- **Worker** — owns a single deploy's state machine: clone → checkout → strip ports → start infra → build → health check → swap → cleanup
- **Refresher** — a heartbeat goroutine that bumps the deploy lock's `expires_at` while the build runs; stops on crash
- **Recovery scanner** — finds deploys whose lock expired (agent died mid-build) and resets them to pending

**Dashboard** — a Next.js App Router frontend. The two-slot view derives "live" from the service's `active_slot`; deploy history and slot state poll live while any build is in flight. Rollback is a single POST that flips nginx and swaps the slots.

---

## The boundary that makes it safe

**You cannot blue/green a database.** There is one database, shared by both slots.

So Launchpad manages exactly one thing: your **app container**. During a deploy both the old and new slots are briefly alive against that one shared database — so anything stateful must survive the swap, not be part of it. Db, cache, and queues run under a stable project name and persist untouched across every deploy; only the stateless app is swapped per slot.

This is the twelve-factor "attached resources" model — the same separation Heroku enforces. It's what lets you get zero-downtime rollouts without ever risking your data.

---

## Tech stack

| Layer | Choice | Why |
|---|---|---|
| Language | Go 1.25 | Goroutines for the scheduler/worker/refresher concurrency model; single static binary |
| Database | PostgreSQL 16 | Row locks to serialise concurrent webhooks; deploy state machine + expiring locks |
| SQL | sqlc | Type-safe Go from SQL, no ORM |
| Migrations | plain SQL | Run by Postgres on first init via `docker-entrypoint-initdb.d` |
| HTTP router | stdlib `net/http` | Method-pattern routing (Go 1.22+) — no dependency needed |
| Remote exec | `golang.org/x/crypto/ssh` | Runs the build on the customer server; one connection reused per deploy |
| Secrets | AES-256-GCM (`pkg/crypto`) | SSH private keys encrypted at rest with `ENCRYPTION_KEY` |
| Frontend | Next.js 16 (App Router) | Client component polls deploy + service state live during a build |
| Styling | Tailwind CSS v4 | Dark theme, CSS variables, the 3D slot cards and grid texture |
| Fonts | Anton / Oswald | Condensed display faces for service headers and the sidebar |

---

## Getting started

### Prerequisites

- Go 1.25+
- Docker + Docker Compose (for the server and Postgres)
- Node.js 20+ (for the dashboard)
- A customer server reachable over SSH, running Docker and nginx (the deploy target)

### 1. Configure

```bash
cp .env.example .env
```

Set `ENCRYPTION_KEY` — generate one with:

```bash
openssl rand -hex 32
```

`BASE_URL` should be the publicly reachable URL of the Launchpad server (e.g. an ngrok tunnel) so git providers can reach the webhook endpoint.

### 2. Start the server + database

```bash
docker compose up -d --build
```

Postgres runs the migrations in `db/migrations/` on first init. The API comes up on port `8090` (mapped from the container's `8080`).

### 3. Start the dashboard

```bash
cd web
npm install
npm run dev   # http://localhost:3010
```

### 4. Register a service

```bash
curl -X POST http://localhost:8090/services \
  -H "Content-Type: application/json" \
  -d '{
    "name": "relay",
    "repo_url": "https://github.com/you/relay.git",
    "domain": "relay.example.com",
    "health_check_url": "/health",
    "webhook_secret": "your-git-webhook-secret",
    "host": "1.2.3.4",
    "ssh_user": "ubuntu",
    "ssh_private_key": "-----BEGIN OPENSSH PRIVATE KEY-----\n...",
    "blue_port": 4001,
    "green_port": 4002,
    "container_port": 8080,
    "compose_service": "app"
  }'
```

The response includes a webhook URL. Point your git provider's push webhook at it.

### 5. Push to deploy

Every push to the registered repo now triggers a zero-downtime deploy. Watch it land in the dashboard.

---

## REST API

| Method | Path | Description |
|---|---|---|
| `POST` | `/services` | Register a service (SSH key is encrypted at rest) |
| `GET` | `/services` | List all services |
| `GET` | `/services/{id}` | Get one service (with active slot) |
| `GET` | `/services/{serviceID}/deploys` | Deploy history for a service |
| `POST` | `/services/{serviceID}/rollback` | Flip nginx to the standby slot |
| `GET` | `/deploys/{deployID}` | Get a single deploy |
| `POST` | `/webhooks/{serviceID}` | Git push webhook — enqueues a deploy |

---

## Project structure

```
06-launchpad/
├── cmd/
│   └── main.go                 # Server main + wiring, starts the deploy engine
├── db/
│   ├── migrations/             # Plain SQL, run by Postgres on init (001–005)
│   └── queries/                # sqlc input SQL (services.sql, deploys.sql)
├── internal/
│   ├── agent/                  # The deploy engine
│   │   ├── coordinator.go      # Agent struct, dependency contracts, active-worker map
│   │   ├── scheduler.go        # Polls pending deploys, dispatches one worker per service
│   │   ├── worker.go           # A single deploy's state machine (clone → swap → cleanup)
│   │   └── recovery.go         # Resets deploys whose lock expired (crashed mid-build)
│   ├── api/                    # HTTP router + service/deploy/webhook handlers
│   ├── adapters/               # SSH executor adapter
│   ├── config/                 # env-based config
│   ├── deploy/
│   │   ├── domain/             # Deploy, DeployStatus, Slot state machine
│   │   ├── infra/              # sqlc repository impl
│   │   └── usecases/           # create, activate, rollback, updatestatus,
│   │                           #   refreshlock, startuprecovery, recoverybuild, ...
│   ├── service/
│   │   ├── domain/             # Service, Slot
│   │   ├── infra/              # sqlc repository impl
│   │   └── usecases/           # create, get, list, update
│   └── shared/
│       ├── db/                 # pgx pool
│       └── ssh/                # SSH client + executor factory
├── pkg/
│   ├── crypto/                 # AES-256-GCM for SSH keys at rest
│   ├── logger/                 # slog-based structured logger
│   └── result/                 # Result[T] for explicit error handling
├── screenshots/                # UI screenshots + zero-downtime test output
└── web/                        # Next.js 16 dashboard
    ├── app/
    │   └── page.tsx            # Sidebar + selected service detail
    ├── components/
    │   ├── service-sidebar.tsx # Service list, collapse toggle
    │   ├── service-detail.tsx  # Config panel + owns service refetch
    │   ├── deploy-list.tsx     # Two slot cards, rollback, live-polling history
    │   ├── status-badge.tsx    # Status + slot badges
    │   └── create-service-modal.tsx
    └── lib/
        └── api.ts              # Typed fetch helpers + domain types
```

---

## Design notes

**Why the app container only, and never the database?**
During a swap both slots are alive at once, both talking to the one shared database. If a deploy also swapped or migrated the db in a breaking way, the still-running old slot would break — reintroducing the downtime you were trying to kill, and risking your data. Launchpad's scope is deliberately the stateless app; stateful services are attached resources that persist across deploys. Scope is the safety feature.

**Why strip the app's host-port binding on every deploy?**
Docker Compose *merges* port lists in overrides, it never replaces them. If the app's compose file hardcodes `8080:8080`, both blue and green would try to bind host port 8080 and the second slot would fail. Launchpad rewrites only the app service's port block in an ephemeral clone of the compose file (never the source repo, never other services) so each slot binds only its own slot port.

**Why a dead-man's-switch lock instead of a timeout?**
A build can take 30 seconds or 10 minutes — a fixed timeout makes fast builds pay for slow ones, and a crashed agent leaves a deploy stuck in `building` forever. Instead the worker sets `expires_at` on the lock and a refresher goroutine bumps it forward while the build makes progress. Crash → refreshes stop → the lock lapses → the recovery scanner resets the deploy to pending. The heartbeat proves liveness; it doesn't guess a duration.

**Why deploy infra and app separately?**
Supporting services start under a stable project name (`{service}`) with `--scale {app}=0`, so the db/cache come up once and stay up, idempotently, across every deploy. The app is then deployed per-slot (`{service}-blue` / `{service}-green`) with `--no-deps`, connected to the shared infra network so it can still reach the db by hostname. Infra is never restarted mid-deploy.

**Why derive "live" from the service, not the deploy status?**
`deploy.status = active` is stamped at activation and isn't cleared on the other slot when nginx flips. The authoritative source of which slot is live is the service's `active_slot`, updated atomically with the nginx switch. The dashboard reads that, so the LIVE/STANDBY cards can't disagree with reality after a rollback.

---

## What's not built yet

- **Schema migrations** — Launchpad doesn't run db migrations; for now, keep them backward-compatible (expand/contract) so both slots can run against the shared db during a swap
- **Automatic rollback** — rollback is manual; there's no post-swap error-rate monitor to trigger it automatically yet
- **Webhook signature verification** — the webhook secret is stored but not yet enforced on incoming payloads
- **Log streaming** — build logs aren't streamed to the dashboard in real time; status transitions are, but not raw `docker` output
- **Multi-host services** — one service deploys to one host; no load-balancing a service across several customer servers
