-- name: GetDeployByID :one
SELECT id, service_id, slot, status, commit_sha, commit_message, pushed_at,
       started_at, finished_at, created_at
FROM deploys WHERE id = $1;

-- name: SetDeployBuilding :exec
UPDATE deploys SET status = 'building', slot = $2, started_at = NOW() WHERE id = $1;

-- name: SetDeployTerminal :exec
UPDATE deploys SET status = $2, finished_at = NOW() WHERE id = $1;

-- name: CreateDeployLock :exec
INSERT INTO deploy_locks (deploy_id, expires_at) VALUES ($1, $2);

-- name: ReleaseDeployLock :exec
UPDATE deploy_locks SET released_at = NOW() WHERE deploy_id = $1 AND released_at IS NULL;

-- name: LockServiceRow :one
-- Acquires a row-level lock on the service to serialise concurrent deploy enqueues.
SELECT id FROM services WHERE id = $1 FOR UPDATE;

-- name: GetPendingDeploy :one
SELECT id, service_id, slot, status, commit_sha, commit_message, pushed_at,
       started_at, finished_at, created_at
FROM deploys
WHERE service_id = $1 AND status = 'pending'
LIMIT 1;

-- name: InsertDeploy :one
INSERT INTO deploys (id, service_id, commit_sha, commit_message, pushed_at, status)
VALUES ($1, $2, $3, $4, $5, 'pending')
RETURNING id, service_id, slot, status, commit_sha, commit_message, pushed_at,
          started_at, finished_at, created_at;

-- name: ListPendingDeploys :many
SELECT id, service_id, slot, status, commit_sha, commit_message, pushed_at,
       started_at, finished_at, created_at
FROM deploys
WHERE status = 'pending'
ORDER BY created_at ASC;

-- name: ListDeploysByService :many
SELECT id, service_id, slot, status, commit_sha, commit_message, pushed_at,
       started_at, finished_at, created_at
FROM deploys
WHERE service_id = $1
ORDER BY created_at DESC;

-- name: GetActiveDeployForService :one
SELECT id, service_id, slot, status, commit_sha, commit_message, pushed_at,
       started_at, finished_at, created_at
FROM deploys
WHERE service_id = $1 AND status = 'active'
LIMIT 1;

-- name: GetLatestDeployOnSlot :one
SELECT id, service_id, slot, status, commit_sha, commit_message, pushed_at,
       started_at, finished_at, created_at
FROM deploys
WHERE service_id = $1 AND slot = $2
ORDER BY created_at DESC
LIMIT 1;

-- name: RefreshDeployLock :exec
UPDATE deploy_locks SET expires_at = $2 WHERE deploy_id = $1 AND released_at IS NULL;

-- name: UpgradePendingDeploy :one
UPDATE deploys
SET commit_sha = $2, commit_message = $3, pushed_at = $4
WHERE id = $1
RETURNING id, service_id, slot, status, commit_sha, commit_message, pushed_at,
          started_at, finished_at, created_at;
