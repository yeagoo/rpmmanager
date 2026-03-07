#!/usr/bin/env bash
set -euo pipefail

# ── Configuration ──────────────────────────────────────────────────
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMPOSE_FILE="$SCRIPT_DIR/docker-compose.yml"
COMPOSE_PROJECT="inttest"

API_URL="http://localhost:18080"
REPO_URL="http://localhost:18081"
TEST_VERSION="0.1.0"
POLL_INTERVAL=3
POLL_TIMEOUT=300

# Detect host architecture (RPM naming: x86_64 / aarch64)
HOST_ARCH="$(uname -m)"

# ── Colors ─────────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
BOLD='\033[1m'
NC='\033[0m'

PASS=0
FAIL=0
TESTS=()

# ── Helpers ────────────────────────────────────────────────────────
log()  { echo -e "${BLUE}[TEST]${NC} $*"; }
pass() { echo -e "${GREEN}[PASS]${NC} $*"; PASS=$((PASS+1)); TESTS+=("PASS: $*"); }
fail() { echo -e "${RED}[FAIL]${NC} $*"; FAIL=$((FAIL+1)); TESTS+=("FAIL: $*"); }

cleanup() {
    log "Cleaning up..."
    for c in inttest-alma8 inttest-alma9 inttest-rocky9; do
        docker rm -f "$c" 2>/dev/null || true
    done
    docker compose -p "$COMPOSE_PROJECT" -f "$COMPOSE_FILE" down -v --remove-orphans 2>/dev/null || true
}

api() {
    local method="$1" path="$2"
    shift 2
    curl -sf -X "$method" \
        -H "X-API-Token: $API_TOKEN" \
        -H "Content-Type: application/json" \
        "$API_URL$path" "$@"
}

wait_for_health() {
    log "Waiting for rpmmanager to be healthy..."
    local elapsed=0
    while [ $elapsed -lt 120 ]; do
        if curl -sf "$API_URL/api/health" > /dev/null 2>&1; then
            log "rpmmanager is healthy"
            return 0
        fi
        sleep 2
        elapsed=$((elapsed+2))
    done
    fail "rpmmanager did not become healthy within 120s"
    return 1
}

poll_build() {
    local build_id="$1"
    local elapsed=0
    while [ $elapsed -lt $POLL_TIMEOUT ]; do
        local status
        status=$(api GET "/api/builds/$build_id" | jq -r '.status')
        case "$status" in
            success)  return 0 ;;
            failed|cancelled)
                log "Build $build_id status: $status"
                api GET "/api/builds/$build_id/log" 2>/dev/null || true
                return 1
                ;;
        esac
        sleep "$POLL_INTERVAL"
        elapsed=$((elapsed+POLL_INTERVAL))
    done
    fail "Build $build_id timed out after ${POLL_TIMEOUT}s"
    return 1
}

test_distro_install() {
    local image="$1" container_name="$2" el_version="$3"
    log "  Testing RPM install on $image..."

    docker run -d --name "$container_name" --network inttest "$image" sleep 300 > /dev/null

    docker cp "$SCRIPT_DIR/testdata/verify/install-rpm.sh" "$container_name:/tmp/install-rpm.sh"

    local exit_code=0
    docker exec "$container_name" bash /tmp/install-rpm.sh \
        "http://repo" "testapp" "$el_version" "$HOST_ARCH" "$TEST_VERSION" \
        || exit_code=$?

    if [ $exit_code -eq 0 ]; then
        pass "RPM install + run on $image"
    else
        fail "RPM install + run on $image (exit code: $exit_code)"
    fi

    docker rm -f "$container_name" > /dev/null 2>&1
}

# ── Main ──────────────────────────────────────────────────────────

trap cleanup EXIT

echo ""
echo -e "${BOLD}${BLUE}=========================================="
echo "  RPM Manager Integration Tests"
echo -e "==========================================${NC}"
echo ""

# ── Phase 0: Setup ────────────────────────────────────────────────

export API_TOKEN="inttest-token-$(date +%s)"
export RPMMANAGER_AUTH_API_TOKEN="$API_TOKEN"

log "Starting Docker Compose stack..."
docker compose -p "$COMPOSE_PROJECT" -f "$COMPOSE_FILE" build --quiet
docker compose -p "$COMPOSE_PROJECT" -f "$COMPOSE_FILE" up -d

wait_for_health || exit 1

# ── Phase 1: Health check ────────────────────────────────────────

log "Phase 1: Health check"
health=$(curl -sf "$API_URL/api/health" | jq -r '.status')
if [ "$health" = "ok" ]; then
    pass "Health check returns ok"
else
    fail "Health check returned: $health"
    exit 1
fi

# ── Phase 2: Generate GPG key ────────────────────────────────────

log "Phase 2: GPG key generation"
gpg_response=$(api POST "/api/gpg-keys/generate" -d '{
    "name": "Integration Test",
    "email": "test@example.com",
    "algorithm": "RSA",
    "key_length": 2048,
    "expire": "0"
}')

GPG_KEY_ID=$(echo "$gpg_response" | jq -r '.id')
if [ -n "$GPG_KEY_ID" ] && [ "$GPG_KEY_ID" != "null" ]; then
    pass "GPG key generated (id=$GPG_KEY_ID)"
else
    fail "GPG key generation failed: $gpg_response"
    exit 1
fi

api POST "/api/gpg-keys/$GPG_KEY_ID/default" > /dev/null
pass "GPG key set as default"

# ── Phase 3: Import test product ─────────────────────────────────

log "Phase 3: Product import (arch=$HOST_ARCH)"
import_payload=$(sed "s/x86_64/$HOST_ARCH/g" "$SCRIPT_DIR/testdata/products/test-simple.json")
import_response=$(api POST "/api/products/import" -d "$import_payload")
PRODUCT_ID=$(echo "$import_response" | jq -r '.imported[0].id')

if [ -n "$PRODUCT_ID" ] && [ "$PRODUCT_ID" != "null" ]; then
    pass "Product imported (id=$PRODUCT_ID, name=testapp)"
else
    fail "Product import failed: $import_response"
    exit 1
fi

# Get current product data, set GPG key, and update
product_data=$(api GET "/api/products/$PRODUCT_ID")
update_payload=$(echo "$product_data" | jq ".gpg_key_id = $GPG_KEY_ID")
update_response=$(api PUT "/api/products/$PRODUCT_ID" -d "$update_payload")
updated_gpg=$(echo "$update_response" | jq -r '.gpg_key_id')
if [ "$updated_gpg" = "$GPG_KEY_ID" ]; then
    pass "GPG key assigned to product"
else
    fail "GPG key assignment failed"
fi

# ── Phase 4: Trigger build ───────────────────────────────────────

log "Phase 4: Build trigger and polling"
build_response=$(api POST "/api/builds" -d "{
    \"product_id\": $PRODUCT_ID,
    \"version\": \"$TEST_VERSION\"
}")
BUILD_ID=$(echo "$build_response" | jq -r '.id')

if [ -n "$BUILD_ID" ] && [ "$BUILD_ID" != "null" ]; then
    pass "Build triggered (id=$BUILD_ID)"
else
    fail "Build trigger failed: $build_response"
    exit 1
fi

if poll_build "$BUILD_ID"; then
    build_detail=$(api GET "/api/builds/$BUILD_ID")
    rpm_count=$(echo "$build_detail" | jq -r '.rpm_count')
    duration=$(echo "$build_detail" | jq -r '.duration_seconds')
    pass "Build completed (rpms=$rpm_count, duration=${duration}s)"
else
    fail "Build did not complete successfully"
    log "Build log:"
    api GET "/api/builds/$BUILD_ID/log" 2>/dev/null || true
    exit 1
fi

# ── Phase 5: Verify repo structure ───────────────────────────────

log "Phase 5: Verify repo structure"
sleep 2

if curl -sf "$REPO_URL/testapp/gpg.key" | head -1 | grep -q "BEGIN PGP PUBLIC KEY BLOCK"; then
    pass "gpg.key exists and is valid"
else
    fail "gpg.key not found or invalid"
fi

if curl -sf "$REPO_URL/testapp/el9/$HOST_ARCH/repodata/repomd.xml" | grep -q "repomd"; then
    pass "el9/$HOST_ARCH repodata exists"
else
    fail "el9/$HOST_ARCH repodata not found"
fi

if curl -sf "$REPO_URL/testapp/el8/$HOST_ARCH/repodata/repomd.xml" | grep -q "repomd"; then
    pass "el8/$HOST_ARCH repodata exists"
else
    fail "el8/$HOST_ARCH repodata not found"
fi

# ── Phase 6: RPM installation on real distros ────────────────────

log "Phase 6: RPM installation tests"

test_distro_install "almalinux:8" "inttest-alma8" "el8"
test_distro_install "almalinux:9" "inttest-alma9" "el9"
test_distro_install "rockylinux:9" "inttest-rocky9" "el9"

# ── Summary ──────────────────────────────────────────────────────

echo ""
echo -e "${BOLD}=========================================="
echo "  Results: $PASS passed, $FAIL failed"
echo -e "==========================================${NC}"
for t in "${TESTS[@]}"; do
    if [[ "$t" == PASS:* ]]; then
        echo -e "  ${GREEN}$t${NC}"
    else
        echo -e "  ${RED}$t${NC}"
    fi
done
echo ""

if [ $FAIL -gt 0 ]; then
    exit 1
fi
