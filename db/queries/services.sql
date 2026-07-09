-- name: SaveService :exec
INSERT INTO services (id, name, repo_url, domain, health_check_url, webhook_secret, host, ssh_user, ssh_key_path, blue_port, green_port, container_port, active_slot, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NOW());

-- name: GetService :one
SELECT id, name, repo_url, domain, health_check_url, webhook_secret, host, ssh_user, ssh_key_path, blue_port, green_port, container_port, active_slot, created_at
FROM services
WHERE id = $1;

-- name: ListServices :many
SELECT id, name, repo_url, domain, health_check_url, webhook_secret, host, ssh_user, ssh_key_path, blue_port, green_port, container_port, active_slot, created_at
FROM services
ORDER BY created_at DESC;

-- name: ExistsByDomain :one
SELECT EXISTS (
    SELECT 1 FROM services WHERE domain = $1
) AS exists;

-- name: DeleteService :exec
DELETE FROM services WHERE id = $1;

-- name: UpdateService :exec
UPDATE services
SET name = $2, health_check_url = $3
WHERE id = $1;

-- name: UpdateServiceActiveSlot :exec
UPDATE services SET active_slot = $2 WHERE id = $1;
