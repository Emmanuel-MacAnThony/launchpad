# Launchpad — Design Challenges

## 1. Deploy Lock — Row vs Transaction
Two approaches to ensuring only one deploy runs per service at a time.
- **Row-based lock**: INSERT a lock row, DELETE when done. Simple but if agent crashes between INSERT and DELETE, lock stays forever → service permanently blocked. Needs expiry as safety net.
- **Transaction-based lock**: Lock lives inside a DB transaction. Crash = rollback = automatic release. No expiry needed.
- **Challenge**: Which approach and why. What are the failure modes of each.

## 2. Agent Decomposition
Currently modelled as one "agent" doing everything. Needs to be broken down.
- **Controller**: validates webhook, creates deploy record, acquires lock, hands off
- **Worker**: orchestrates the full deploy state machine (clone → build → start → health check → swap → monitor → cleanup)
- **Host Agent**: tiny HTTP server running on target server, exposes endpoints for build/start/stop/logs, eliminates SSH scripting
- **Challenge**: where does each responsibility live, how do they communicate, what happens when one fails

## 3. Deploy State Machine
Deploy status is not just a field — it's a state machine with explicit transitions and failure states.
- States: QUEUED → BUILDING → STARTING → HEALTH_CHECKING → READY → SWAPPING → ACTIVE → MONITORING → COMPLETED
- Failure states: BUILD_FAILED, START_FAILED, HEALTH_FAILED, ROLLBACK, ROLLED_BACK
- **Challenge**: model every valid transition, what triggers each, what happens on invalid transitions

## 4. Log Streaming
Developer needs to watch deploy progress in real time.
- Browser ←── WS ──→ Launchpad Server ←── WS ──→ Host Agent
- Host agent streams docker logs, Launchpad pipes to browser
- **Challenge**: connection lifecycle (what happens if browser disconnects mid-deploy, what if host agent drops), backpressure, buffering logs for reconnect

## 5. Host Agent Installation
Host agent needs to run permanently on every target server.
- Installed once at service registration via SSH + install script
- Runs as a systemd service
- **Challenge**: versioning (how do you update the agent), authentication (how does Launchpad verify it's talking to a legitimate agent and not a rogue server), what happens if agent is unreachable mid-deploy

## 6. Health Check Gate
Not just a single 200 — needs to be robust.
- 3 consecutive 200s within 60s timeout
- Response latency under 500ms
- **Challenge**: what counts as a failure (timeout, 5xx, slow response), how do you handle flapping (pass, fail, pass), when do you give up and mark deploy as HEALTH_FAILED

## 7. Rollback
Rollback is a new deploy pointing at the previous commit.
- Standby container kept alive after swap for rollback window
- Triggered automatically (error spike post-swap) or manually
- `rollback_of` field on Deploy links it to the deploy it reverted
- **Challenge**: what triggers automatic rollback, how do you detect an error spike, what if standby container died during the rollback window

## 8. Nginx Config Management
Agent generates nginx config at promotion time, not at request time.
- Templates upstream block with active container port
- Runs `nginx -s reload` — graceful, never drops connections
- **Challenge**: config file management (one file per service or one big file), what happens if nginx reload fails, config drift if someone manually edits nginx
