-- name: GetDeployByID :one
SELECT id, service_id, slot, status, commit_sha, commit_message, pushed_at,
       started_at, finished_at, created_at
FROM deploys WHERE id = $1;

-- name: SetDeployBuilding :exec
UPDATE deploys SET status = 'building', slot = $2, started_at = NOW() WHERE id = $1;

-- name: SetDeployTerminal :exec
UPDATE deploys SET status = $2, finished_at = NOW() WHERE id = $1;

-- name: CreateDeployLock :exec
INSERT INTO deploy_locks (deploy_id, expires_at) VALUES ($1, $2)
ON CONFLICT (deploy_id) DO UPDATE
  SET locked_at = NOW(), expires_at = EXCLUDED.expires_at, released_at = NULL;

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

-- name: GetLatestPushedAt :one
SELECT pushed_at FROM deploys
WHERE service_id = $1
  AND status IN ('pending', 'building', 'active')
ORDER BY pushed_at DESC
LIMIT 1;

-- name: StartupRecoveryReleaseLocks :exec
-- Release locks held by any building deploy (process crashed; all building are stale).
UPDATE deploy_locks SET released_at = NOW()
WHERE deploy_id IN (SELECT id FROM deploys WHERE status = 'building')
  AND released_at IS NULL;

-- name: StartupRecoveryResetBuilding :execresult
UPDATE deploys SET status = 'pending', slot = NULL, started_at = NULL
WHERE status = 'building';

-- name: ResetExpiredBuilding :execresult
-- Atomically release expired locks and reset their deploys to pending.
WITH expired AS (
    SELECT deploy_id FROM deploy_locks
    WHERE expires_at < NOW() AND released_at IS NULL
),
release AS (
    UPDATE deploy_locks SET released_at = NOW()
    WHERE deploy_id IN (SELECT deploy_id FROM expired)
    RETURNING deploy_id
)
UPDATE deploys SET status = 'pending', slot = NULL, started_at = NULL
WHERE id IN (SELECT deploy_id FROM release)
  AND status = 'building';

-- name: UpgradePendingDeploy :one
UPDATE deploys
SET commit_sha = $2, commit_message = $3, pushed_at = $4
WHERE id = $1
RETURNING id, service_id, slot, status, commit_sha, commit_message, pushed_at,
          started_at, finished_at, created_at;
