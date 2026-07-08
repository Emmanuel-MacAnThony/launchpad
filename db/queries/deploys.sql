-- name: LockServiceRow :one
-- Acquires a row-level lock on the service to serialise concurrent deploy enqueues.
SELECT id FROM services WHERE id = $1 FOR UPDATE;

-- name: GetPendingDeploy :one
SELECT id, service_id, slot, status, commit_sha, commit_message, pushed_at,
       rollback_of, started_at, finished_at, created_at
FROM deploys
WHERE service_id = $1 AND status = 'pending'
LIMIT 1;

-- name: InsertDeploy :one
INSERT INTO deploys (id, service_id, commit_sha, commit_message, pushed_at, status)
VALUES ($1, $2, $3, $4, $5, 'pending')
RETURNING id, service_id, slot, status, commit_sha, commit_message, pushed_at,
          rollback_of, started_at, finished_at, created_at;

-- name: ListPendingDeploys :many
SELECT id, service_id, slot, status, commit_sha, commit_message, pushed_at,
       rollback_of, started_at, finished_at, created_at
FROM deploys
WHERE status = 'pending'
ORDER BY created_at ASC;

-- name: UpgradePendingDeploy :one
UPDATE deploys
SET commit_sha = $2, commit_message = $3, pushed_at = $4
WHERE id = $1
RETURNING id, service_id, slot, status, commit_sha, commit_message, pushed_at,
          rollback_of, started_at, finished_at, created_at;
