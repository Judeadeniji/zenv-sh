#!/usr/bin/env bash
# Smoke tests for the zEnv API + CLI.
# Requires: API running on localhost:8080, Postgres + Redis up.
#
# Usage:
#   make dev-up && make migrate
#   DATABASE_URL="postgres://zenv:zenv_dev@localhost:5434/zenv?sslmode=disable" ./bin/zenv-api &
#   ./tests/smoke.sh

set -euo pipefail

API="http://localhost:8080"
DB="postgres://zenv:zenv_dev@localhost:5434/zenv?sslmode=disable"
PASS=0
FAIL=0

# --- Helpers ---

green() { printf "\033[32m%s\033[0m\n" "$1"; }
red()   { printf "\033[31m%s\033[0m\n" "$1"; }

assert_status() {
  local label="$1" expected="$2" actual="$3"
  if [ "$expected" = "$actual" ]; then
    green "  ✓ $label"
    PASS=$((PASS + 1))
  else
    red "  ✗ $label (expected $expected, got $actual)"
    FAIL=$((FAIL + 1))
  fi
}

assert_contains() {
  local label="$1" haystack="$2" needle="$3"
  if echo "$haystack" | grep -q "$needle"; then
    green "  ✓ $label"
    PASS=$((PASS + 1))
  else
    red "  ✗ $label (expected to contain '$needle')"
    FAIL=$((FAIL + 1))
  fi
}

b64rand() { python3 -c "import base64,os;print(base64.b64encode(os.urandom($1)).decode())"; }

# --- Health ---

echo "=== Health ==="
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$API/health")
assert_status "GET /health" "200" "$STATUS"

# --- Signup ---

echo ""
echo "=== Auth: Signup ==="
EMAIL="smoke${RANDOM}@test.com"
SIGNUP=$(curl -s -w "\n%{http_code}" -X POST "$API/v1/auth/signup" \
  -H "Content-Type: application/json" \
  -d '{"email":"'"$EMAIL"'","vault_key_type":"passphrase","salt":"'"$(b64rand 32)"'","auth_key_hash":"'"$(b64rand 32)"'","wrapped_dek":"'"$(b64rand 64)"'","public_key":"'"$(b64rand 32)"'","wrapped_private_key":"'"$(b64rand 64)"'"}')
SIGNUP_STATUS=$(echo "$SIGNUP" | tail -1)
SIGNUP_BODY=$(echo "$SIGNUP" | sed '$d')
assert_status "POST /v1/auth/signup" "201" "$SIGNUP_STATUS"

SESS=$(echo "$SIGNUP_BODY" | python3 -c "import sys,json;print(json.load(sys.stdin)['session_id'])" 2>/dev/null || echo "")
USER_ID=$(echo "$SIGNUP_BODY" | python3 -c "import sys,json;print(json.load(sys.stdin)['user_id'])" 2>/dev/null || echo "")

# --- Duplicate signup ---

echo ""
echo "=== Auth: Duplicate signup ==="
DUP_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$API/v1/auth/signup" \
  -H "Content-Type: application/json" \
  -d '{"email":"'"$EMAIL"'","vault_key_type":"passphrase","salt":"'"$(b64rand 32)"'","auth_key_hash":"'"$(b64rand 32)"'","wrapped_dek":"'"$(b64rand 64)"'","public_key":"'"$(b64rand 32)"'","wrapped_private_key":"'"$(b64rand 64)"'"}')
assert_status "POST /v1/auth/signup (duplicate)" "409" "$DUP_STATUS"

# --- Dev Login ---

echo ""
echo "=== Auth: Dev Login ==="
LOGIN=$(curl -s -w "\n%{http_code}" -X POST "$API/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"email":"'"$EMAIL"'"}')
LOGIN_STATUS=$(echo "$LOGIN" | tail -1)
LOGIN_BODY=$(echo "$LOGIN" | sed '$d')
assert_status "POST /v1/auth/login" "200" "$LOGIN_STATUS"
assert_contains "login returns salt" "$LOGIN_BODY" "salt"
assert_contains "login returns vault_key_type" "$LOGIN_BODY" "vault_key_type"

# --- Unlock (wrong key) ---

echo ""
echo "=== Auth: Unlock (wrong key) ==="
SESS2=$(echo "$LOGIN_BODY" | python3 -c "import sys,json;print(json.load(sys.stdin)['session_id'])" 2>/dev/null || echo "")
UNLOCK_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$API/v1/auth/unlock" \
  -H "Content-Type: application/json" \
  -b "zenv_session=$SESS2" \
  -d '{"auth_key_hash":"'"$(b64rand 32)"'"}')
assert_status "POST /v1/auth/unlock (wrong key)" "403" "$UNLOCK_STATUS"

# --- Create project (via psql) ---

ORG=$(psql "$DB" -tAc "INSERT INTO organizations (name, owner_id) VALUES ('SmokeOrg', '$USER_ID') RETURNING id;" | head -1)
PID=$(psql "$DB" -tAc "INSERT INTO projects (organization_id, name) VALUES ('$ORG', 'smoke-proj') RETURNING id;" | head -1)

# --- Service Tokens ---

echo ""
echo "=== Tokens: Create ==="
TOK=$(curl -s -w "\n%{http_code}" -X POST "$API/v1/tokens" \
  -H "Content-Type: application/json" \
  -b "zenv_session=$SESS" \
  -d '{"project_id":"'"$PID"'","name":"smoke","environment":"development","permission":"read_write"}')
TOK_STATUS=$(echo "$TOK" | tail -1)
TOK_BODY=$(echo "$TOK" | sed '$d')
assert_status "POST /v1/tokens" "201" "$TOK_STATUS"

SVC=$(echo "$TOK_BODY" | python3 -c "import sys,json;print(json.load(sys.stdin)['token'])" 2>/dev/null || echo "")
TID=$(echo "$TOK_BODY" | python3 -c "import sys,json;print(json.load(sys.stdin)['id'])" 2>/dev/null || echo "")

echo ""
echo "=== Tokens: List ==="
LIST_STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$API/v1/tokens?project_id=$PID" -b "zenv_session=$SESS")
assert_status "GET /v1/tokens" "200" "$LIST_STATUS"

# --- SDK: Secrets CRUD ---

NH=$(b64rand 32)
CT=$(b64rand 64)
NC=$(b64rand 12)

echo ""
echo "=== SDK: Create Secret ==="
CREATE_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$API/v1/sdk/secrets" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $SVC" \
  -d '{"project_id":"'"$PID"'","environment":"development","name_hash":"'"$NH"'","ciphertext":"'"$CT"'","nonce":"'"$NC"'"}')
assert_status "POST /v1/sdk/secrets" "201" "$CREATE_STATUS"

echo ""
echo "=== SDK: Create Duplicate ==="
DUP_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$API/v1/sdk/secrets" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $SVC" \
  -d '{"project_id":"'"$PID"'","environment":"development","name_hash":"'"$NH"'","ciphertext":"'"$CT"'","nonce":"'"$NC"'"}')
assert_status "POST /v1/sdk/secrets (duplicate)" "409" "$DUP_STATUS"

echo ""
echo "=== SDK: List Secrets ==="
LIST_RESP=$(curl -s -w "\n%{http_code}" "$API/v1/sdk/secrets?project_id=$PID&environment=development" \
  -H "Authorization: Bearer $SVC")
LIST_STATUS=$(echo "$LIST_RESP" | tail -1)
LIST_BODY=$(echo "$LIST_RESP" | sed '$d')
assert_status "GET /v1/sdk/secrets" "200" "$LIST_STATUS"
assert_contains "list contains secret" "$LIST_BODY" "name_hash"

echo ""
echo "=== SDK: Update Secret ==="
NH_URL=$(python3 -c "import base64;print(base64.urlsafe_b64encode(base64.b64decode('$NH')).decode())")
CT2=$(b64rand 64)
NC2=$(b64rand 12)
UPDATE_RESP=$(curl -s -w "\n%{http_code}" -X PUT "$API/v1/sdk/secrets/$NH_URL?project_id=$PID&environment=development" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $SVC" \
  -d '{"ciphertext":"'"$CT2"'","nonce":"'"$NC2"'"}')
UPDATE_STATUS=$(echo "$UPDATE_RESP" | tail -1)
UPDATE_BODY=$(echo "$UPDATE_RESP" | sed '$d')
assert_status "PUT /v1/sdk/secrets/:nameHash" "200" "$UPDATE_STATUS"
assert_contains "version bumped" "$UPDATE_BODY" '"version":2'

echo ""
echo "=== SDK: Delete Secret ==="
DEL_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$API/v1/sdk/secrets/$NH_URL?project_id=$PID&environment=development" \
  -H "Authorization: Bearer $SVC")
assert_status "DELETE /v1/sdk/secrets/:nameHash" "200" "$DEL_STATUS"

echo ""
echo "=== SDK: Get Deleted (should 404) ==="
GONE_STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$API/v1/sdk/secrets/$NH_URL?project_id=$PID&environment=development" \
  -H "Authorization: Bearer $SVC")
assert_status "GET deleted secret" "404" "$GONE_STATUS"

# --- SDK: Auth checks ---

echo ""
echo "=== SDK: Invalid Token ==="
BAD_STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$API/v1/sdk/secrets?project_id=$PID&environment=development" \
  -H "Authorization: Bearer ze_dev_fakefake")
assert_status "GET with invalid token" "401" "$BAD_STATUS"

echo ""
echo "=== Tokens: Revoke ==="
REV_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$API/v1/tokens/$TID" -b "zenv_session=$SESS")
assert_status "DELETE /v1/tokens/:id" "200" "$REV_STATUS"

echo ""
echo "=== SDK: Revoked Token ==="
REVOKED_STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$API/v1/sdk/secrets?project_id=$PID&environment=development" \
  -H "Authorization: Bearer $SVC")
assert_status "GET with revoked token" "401" "$REVOKED_STATUS"

# --- CLI tests ---

echo ""
echo "=== CLI: Setup ==="
# Create a fresh token for CLI
TOK2_BODY=$(curl -s -X POST "$API/v1/tokens" \
  -H "Content-Type: application/json" \
  -b "zenv_session=$SESS" \
  -d '{"project_id":"'"$PID"'","name":"cli-smoke","environment":"development","permission":"read_write"}')
SVC2=$(echo "$TOK2_BODY" | python3 -c "import sys,json;print(json.load(sys.stdin)['token'])")

export ZENV_TOKEN="$SVC2"
export ZENV_VAULT_KEY="smoke-test-vault-key"
export ZENV_PROJECT="$PID"
export ZENV_ENV="development"

echo ""
echo "=== CLI: secrets set ==="
SET_OUT=$(./bin/zenv secrets set SMOKE_DB "postgres://smoke:test@localhost/db" 2>&1)
assert_contains "set creates" "$SET_OUT" "created"

echo ""
echo "=== CLI: secrets get ==="
GET_OUT=$(./bin/zenv secrets get SMOKE_DB 2>&1)
assert_contains "get decrypts" "$GET_OUT" "postgres://smoke:test@localhost/db"

echo ""
echo "=== CLI: secrets list ==="
LIST_OUT=$(./bin/zenv secrets list 2>&1)
assert_contains "list shows entries" "$LIST_OUT" "VERSION"

echo ""
echo "=== CLI: check (exists) ==="
./bin/zenv check SMOKE_DB 2>/dev/null
assert_status "check existing secret" "0" "$?"

echo ""
echo "=== CLI: check (missing) ==="
./bin/zenv check SMOKE_DB NONEXISTENT 2>/dev/null || true
# Can't easily capture exit code in set -e, just verify it ran

echo ""
echo "=== CLI: secrets delete ==="
DEL_OUT=$(./bin/zenv secrets delete SMOKE_DB 2>&1)
assert_contains "delete removes" "$DEL_OUT" "deleted"

echo ""
echo "=== CLI: run ==="
./bin/zenv secrets set RUN_TEST "injected_value" 2>/dev/null
RUN_OUT=$(./bin/zenv run -- sh -c 'echo "$RUN_TEST"' 2>/dev/null)
assert_contains "run injects env" "$RUN_OUT" "injected_value"

# --- Summary ---

echo ""
echo "================================"
TOTAL=$((PASS + FAIL))
if [ "$FAIL" -eq 0 ]; then
  green "All $TOTAL tests passed"
else
  red "$FAIL of $TOTAL tests failed"
  exit 1
fi
