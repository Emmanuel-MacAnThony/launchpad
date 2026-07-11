#!/usr/bin/env bash
# Hammers the relay service through nginx once per second.
# All requests go through the nginx proxy so slot swaps are visible in real-time.
# Run from anywhere — uses docker exec to reach the container network.
#
# Usage: ./test/zero-downtime/hammer.sh

CONTAINER="06-launchpad-customer-1"
PASS=0
FAIL=0
LAST_SLOT=""
START=$(date +%s)

GREEN='\033[32m'
RED='\033[31m'
YELLOW='\033[33m'
CYAN='\033[36m'
DIM='\033[2m'
BOLD='\033[1m'
RESET='\033[0m'

slot() {
    docker exec 06-launchpad-app-1 wget -q -O- \
        "http://localhost:8080/services/a2bf821e-79c7-4e6e-ad69-78b3ae4032bc" 2>/dev/null \
        | grep -o '"active_slot":"[^"]*"' | cut -d'"' -f4
}

on_exit() {
    ELAPSED=$(( $(date +%s) - START ))
    echo ""
    echo -e "  ${BOLD}─────────────────────────────────────────${RESET}"
    echo -e "  ${GREEN}✓${RESET} ${BOLD}${PASS}${RESET} ok   ${RED}✗${RESET} ${BOLD}${FAIL}${RESET} fail   ${DIM}${ELAPSED}s elapsed${RESET}"
    if [ "$FAIL" -eq 0 ]; then
        echo -e "  ${GREEN}${BOLD}Zero downtime confirmed.${RESET}"
    else
        echo -e "  ${RED}${BOLD}${FAIL} request(s) dropped.${RESET}"
    fi
    echo ""
    exit 0
}

trap on_exit INT TERM

echo ""
echo -e "  ${BOLD}Launchpad zero-downtime test${RESET}"
echo -e "  ${DIM}Hitting relay via nginx · 1 req/s · Ctrl+C to stop${RESET}"
echo -e "  ${BOLD}─────────────────────────────────────────${RESET}"
echo ""

while true; do
    TS=$(date +%H:%M:%S)

    RESULT=$(docker exec "$CONTAINER" \
        curl -s -o /dev/null \
        -w "%{http_code} %{time_total}" \
        -H "Host: relay.local" \
        http://127.0.0.1/health 2>/dev/null)

    CODE=$(echo "$RESULT" | awk '{print $1}')
    TIME_S=$(echo "$RESULT" | awk '{print $2}')
    TIME_MS=$(awk "BEGIN {printf \"%.0f\", ${TIME_S:-0} * 1000}")

    # Detect slot changes
    SLOT=$(slot 2>/dev/null)
    SLOT_TAG=""
    if [ -n "$SLOT" ] && [ "$SLOT" != "$LAST_SLOT" ]; then
        SLOT_TAG="  ${CYAN}← slot: ${SLOT}${RESET}"
        LAST_SLOT="$SLOT"
    fi

    if [ "$CODE" = "200" ]; then
        PASS=$((PASS + 1))
        echo -e "  ${GREEN}✓${RESET}  ${DIM}${TS}${RESET}  ${BOLD}${CODE}${RESET}  ${DIM}${TIME_MS}ms${RESET}${SLOT_TAG}"
    else
        FAIL=$((FAIL + 1))
        echo -e "  ${RED}✗${RESET}  ${TS}  ${RED}${BOLD}${CODE:-ERR}${RESET}  ${TIME_MS}ms  ${RED}← DOWNTIME${RESET}${SLOT_TAG}"
    fi

    sleep 1
done
