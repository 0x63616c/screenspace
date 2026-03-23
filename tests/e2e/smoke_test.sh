#!/bin/bash
# tests/e2e/smoke_test.sh
# Requires: docker compose, curl, jq
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

echo "=== Starting services ==="
cd "$REPO_ROOT/server"
docker compose up -d --build
sleep 5

BASE="http://localhost:8080"

echo "=== Health check ==="
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/health")
[ "$STATUS" = "200" ] || { echo "FAIL: health check returned $STATUS"; exit 1; }

echo "=== Register admin ==="
ADMIN=$(curl -s -X POST "$BASE/api/v1/auth/register" \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"testpassword123"}')
TOKEN=$(echo "$ADMIN" | jq -r '.token')
[ "$TOKEN" != "null" ] || { echo "FAIL: no token"; exit 1; }
ROLE=$(echo "$ADMIN" | jq -r '.role')
[ "$ROLE" = "admin" ] || { echo "FAIL: expected admin role, got $ROLE"; exit 1; }

echo "=== Register user ==="
USER=$(curl -s -X POST "$BASE/api/v1/auth/register" \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"testpassword123"}')
USER_TOKEN=$(echo "$USER" | jq -r '.token')

echo "=== Browse wallpapers (empty) ==="
LIST=$(curl -s "$BASE/api/v1/wallpapers")
echo "$LIST" | jq . > /dev/null || { echo "FAIL: invalid JSON response"; exit 1; }

echo "=== Login ==="
LOGIN=$(curl -s -X POST "$BASE/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"testpassword123"}')
LOGIN_TOKEN=$(echo "$LOGIN" | jq -r '.token')
[ "$LOGIN_TOKEN" != "null" ] || { echo "FAIL: login failed"; exit 1; }

echo "=== Get me ==="
ME=$(curl -s "$BASE/api/v1/auth/me" -H "Authorization: Bearer $TOKEN")
EMAIL=$(echo "$ME" | jq -r '.email')
[ "$EMAIL" = "admin@example.com" ] || { echo "FAIL: wrong email $EMAIL"; exit 1; }

echo "=== All smoke tests passed ==="
docker compose down
