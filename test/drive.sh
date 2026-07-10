#!/bin/bash
set -e

LAUNCHPAD_URL="http://localhost:8090"
WEBHOOK_SECRET="test-webhook-secret"
KEY_DIR="$(dirname "$0")/keys"
PRIVATE_KEY="$KEY_DIR/id_rsa"
PUBLIC_KEY="$KEY_DIR/id_rsa.pub"

# ── 1. SSH keypair ────────────────────────────────────────────────────────────
# Generate once. The public key is mounted into the customer container at
# startup; the private key is mounted into the app container at /keys/id_rsa.

if [ ! -f "$PRIVATE_KEY" ]; then
    echo "Generating SSH keypair..."
    mkdir -p "$KEY_DIR"
    ssh-keygen -t rsa -b 4096 -f "$PRIVATE_KEY" -N "" -C "launchpad-test"
fi

# ── 2. Stack ──────────────────────────────────────────────────────────────────

echo "Building and starting test stack..."
docker compose -f docker-compose.test.yml up -d --build

echo "Waiting for Launchpad API..."
until curl -sf "$LAUNCHPAD_URL/services" > /dev/null 2>&1; do sleep 2; done
echo "API ready"

# ── 3. Register service ───────────────────────────────────────────────────────
# host: "customer" resolves within the docker network.
# ssh_key_path: "/keys/id_rsa" is where the private key is mounted in the app container.
# repo_url: file:// so the agent clones the test app from the customer's local filesystem.

echo "Registering service..."
RESPONSE=$(curl -sf -X POST "$LAUNCHPAD_URL/services" \
    -H "Content-Type: application/json" \
    -d '{
        "name":             "testapp",
        "repo_url":         "file:///home/ubuntu/testapp",
        "domain":           "testapp.local",
        "health_check_url": "http://testapp.local/health",
        "webhook_secret":   "'"$WEBHOOK_SECRET"'",
        "host":             "customer",
        "ssh_user":         "ubuntu",
        "ssh_key_path":     "/keys/id_rsa",
        "blue_port":        3001,
        "green_port":       3002,
        "container_port":   8080
    }')

SERVICE_ID=$(echo "$RESPONSE" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")
echo "Service created: $SERVICE_ID"

# ── 4. Get commit SHA from the customer container ─────────────────────────────
# The agent will checkout this exact SHA during the build step.

COMMIT_SHA=$(docker compose -f docker-compose.test.yml exec -T -u ubuntu customer \
    git -C /home/ubuntu/testapp rev-parse HEAD)
echo "Commit SHA: $COMMIT_SHA"

# ── 5. Build webhook payload and HMAC signature ───────────────────────────────
# Write payload to a temp file so the same bytes go to both openssl and curl.

PUSHED_AT=$(date +%s)
PAYLOAD='{"ref":"refs/heads/main","head_commit":{"id":"'"$COMMIT_SHA"'","message":"Initial commit"},"repository":{"pushed_at":'"$PUSHED_AT"'}}'

PAYLOAD_FILE=$(mktemp)
printf '%s' "$PAYLOAD" > "$PAYLOAD_FILE"

SIGNATURE="sha256=$(openssl dgst -sha256 -hmac "$WEBHOOK_SECRET" < "$PAYLOAD_FILE" | awk '{print $NF}')"
echo "Signature: $SIGNATURE"

# ── 6. Fire webhook ───────────────────────────────────────────────────────────

echo "Firing webhook..."
curl -sf -X POST "$LAUNCHPAD_URL/webhooks/$SERVICE_ID" \
    -H "Content-Type: application/json" \
    -H "X-Hub-Signature-256: $SIGNATURE" \
    --data-binary @"$PAYLOAD_FILE"

rm "$PAYLOAD_FILE"

echo ""
echo "Deploy triggered. Following Launchpad logs (Ctrl-C to stop)..."
docker compose -f docker-compose.test.yml logs -f app
