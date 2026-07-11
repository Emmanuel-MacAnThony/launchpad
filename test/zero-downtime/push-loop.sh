#!/usr/bin/env bash
# Pushes commits to the relay repo at regular intervals to trigger Launchpad deployments.
# Designed to run alongside hammer.sh to demonstrate zero-downtime slot swaps.
#
# Usage: ./test/zero-downtime/push-loop.sh [deploys] [interval_seconds]
#   deploys          — number of deployments to trigger (default: 3)
#   interval_seconds — seconds between each push (default: 90)

DEPLOYS=${1:-3}
INTERVAL=${2:-90}

# Resolve relay project relative to this script's location
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
RELAY_DIR="$(cd "$SCRIPT_DIR/../../../03-relay" && pwd 2>/dev/null)" \
    || { echo "relay project not found at ../../../03-relay"; exit 1; }

BOLD='\033[1m'
GREEN='\033[32m'
CYAN='\033[36m'
DIM='\033[2m'
RESET='\033[0m'

echo ""
echo -e "  ${BOLD}Launchpad push loop${RESET}"
echo -e "  ${DIM}${DEPLOYS} deploys · ${INTERVAL}s apart · relay: ${RELAY_DIR}${RESET}"
echo -e "  ${BOLD}─────────────────────────────────────────${RESET}"
echo ""

for i in $(seq 1 "$DEPLOYS"); do
    echo -e "  ${CYAN}[$(date +%H:%M:%S)]${RESET} Deploy ${BOLD}${i}/${DEPLOYS}${RESET} — pushing..."

    # Write a counter file so every commit has a real diff
    echo "$i" > "$RELAY_DIR/test/deploy-counter.txt"

    cd "$RELAY_DIR" || exit 1
    mkdir -p test
    git add test/deploy-counter.txt
    git commit -m "zero-downtime test: deploy ${i} of ${DEPLOYS}"
    git push origin main

    echo -e "  ${GREEN}✓${RESET} ${DIM}pushed — webhook will trigger deploy${RESET}"

    if [ "$i" -lt "$DEPLOYS" ]; then
        echo -e "  ${DIM}waiting ${INTERVAL}s...${RESET}"
        sleep "$INTERVAL"
    fi
done

echo ""
echo -e "  ${GREEN}${BOLD}All ${DEPLOYS} deploys triggered.${RESET}"
echo ""
