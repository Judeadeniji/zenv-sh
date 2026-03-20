#!/usr/bin/env bash
# Smoke tests for zEnv API + Auth Server + CLI.
# Requires: API running, auth server running, Postgres + Redis up.
#
# Usage:
#   make dev-up && make migrate && make migrate-auth
#   make dev-api &   # API on localhost:8080
#   make dev-auth &  # Auth on localhost:3000
#   ./tests/smoke.sh

set -euo pipefail

API="${ZENV_API_URL:-http://localhost:8080}"
AUTH="${ZENV_AUTH_URL:-http://localhost:3000}"
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
json_field() { python3 -c "import sys,json;print(json.load(sys.stdin)['$1'])" 2>/dev/null; }

# --- Health checks ---

echo "=== Health ==="
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$API/health")
assert_status "GET $API/health" "200" "$STATUS"

STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$AUTH/api/health")
assert_status "GET $AUTH/api/health" "200" "$STATUS"

# --- Auth: Signup via auth server ---

echo ""
echo "=== Auth: Signup ==="
EMAIL="smoke${RANDOM}@test.com"
PASSWORD="smokepw${RANDOM}!"
SIGNUP=$(curl -s -w "\n%{http_code}" -X POST "$AUTH/api/auth/sign-up/email" \
  -H "Content-Type: application/json" \
  -H "Origin: $AUTH" \
  -d '{"name":"Smoke User","email":"'"$EMAIL"'","password":"'"$PASSWORD"'"}')
SIGNUP_STATUS=$(echo "$SIGNUP" | tail -1)
SIGNUP_BODY=$(echo "$SIGNUP" | sed '$d')
assert_status "POST /api/auth/sign-up/email" "200" "$SIGNUP_STATUS"

# Extract session token from response
SESSION_TOKEN=$(echo "$SIGNUP_BODY" | python3 -c "import sys,json;print(json.load(sys.stdin).get('token',''))" 2>/dev/null || echo "")
assert_contains "signup returns token" "$SIGNUP_BODY" "token"

# --- Auth: Duplicate signup ---

echo ""
echo "=== Auth: Duplicate signup ==="
DUP=$(curl -s -w "\n%{http_code}" -X POST "$AUTH/api/auth/sign-up/email" \
  -H "Content-Type: application/json" \
  -H "Origin: $AUTH" \
  -d '{"name":"Smoke User","email":"'"$EMAIL"'","password":"'"$PASSWORD"'"}')
DUP_STATUS=$(echo "$DUP" | tail -1)
# BA returns 200 with error body, or 422 — just verify it doesn't create a duplicate
assert_contains "duplicate rejected" "$(echo "$DUP" | sed '$d')" "already"

# --- Auth: Sign in ---

echo ""
echo "=== Auth: Sign in ==="
SIGNIN=$(curl -s -w "\n%{http_code}" -X POST "$AUTH/api/auth/sign-in/email" \
  -H "Content-Type: application/json" \
  -H "Origin: $AUTH" \
  -d '{"email":"'"$EMAIL"'","password":"'"$PASSWORD"'"}')
SIGNIN_STATUS=$(echo "$SIGNIN" | tail -1)
SIGNIN_BODY=$(echo "$SIGNIN" | sed '$d')
assert_status "POST /api/auth/sign-in/email" "200" "$SIGNIN_STATUS"

SESSION_TOKEN=$(echo "$SIGNIN_BODY" | python3 -c "import sys,json;print(json.load(sys.stdin).get('token',''))" 2>/dev/null || echo "")

# --- API: /auth/me (no vault yet) ---

echo ""
echo "=== API: /auth/me (no vault) ==="
ME=$(curl -s -w "\n%{http_code}" "$API/v1/auth/me" \
  -H "Authorization: Bearer $SESSION_TOKEN")
ME_STATUS=$(echo "$ME" | tail -1)
ME_BODY=$(echo "$ME" | sed '$d')
assert_status "GET /v1/auth/me" "200" "$ME_STATUS"
assert_contains "vault not set up" "$ME_BODY" '"vault_setup_complete":false'

# --- API: Setup vault ---

echo ""
echo "=== API: Setup vault ==="
VAULT=$(curl -s -w "\n%{http_code}" -X POST "$API/v1/auth/setup-vault" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $SESSION_TOKEN" \
  -d '{"vault_key_type":"passphrase","salt":"'"$(b64rand 32)"'","auth_key_hash":"'"$(b64rand 32)"'","wrapped_dek":"'"$(b64rand 64)"'","public_key":"'"$(b64rand 32)"'","wrapped_private_key":"'"$(b64rand 64)"'"}')
VAULT_STATUS=$(echo "$VAULT" | tail -1)
VAULT_BODY=$(echo "$VAULT" | sed '$d')
assert_status "POST /v1/auth/setup-vault" "201" "$VAULT_STATUS"
assert_contains "vault setup complete" "$VAULT_BODY" '"vault_setup_complete":true'

USER_ID=$(echo "$VAULT_BODY" | json_field "user_id" || echo "")

# --- API: /auth/me (vault set up) ---

echo ""
echo "=== API: /auth/me (vault set up) ==="
ME2=$(curl -s -w "\n%{http_code}" "$API/v1/auth/me" \
  -H "Authorization: Bearer $SESSION_TOKEN")
ME2_BODY=$(echo "$ME2" | sed '$d')
assert_contains "vault now set up" "$ME2_BODY" '"vault_setup_complete":true'

# --- API: Duplicate vault setup ---

echo ""
echo "=== API: Duplicate vault setup ==="
DUP_VAULT_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$API/v1/auth/setup-vault" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $SESSION_TOKEN" \
  -d '{"vault_key_type":"passphrase","salt":"'"$(b64rand 32)"'","auth_key_hash":"'"$(b64rand 32)"'","wrapped_dek":"'"$(b64rand 64)"'","public_key":"'"$(b64rand 32)"'","wrapped_private_key":"'"$(b64rand 64)"'"}')
assert_status "POST /v1/auth/setup-vault (duplicate)" "409" "$DUP_VAULT_STATUS"

# --- Create project (via psql for SDK token tests) ---

ORG=$(psql "$DB" -tAc "INSERT INTO organizations (name, owner_id) VALUES ('SmokeOrg-${RANDOM}', '$USER_ID') RETURNING id;" | head -1)
PID=$(psql "$DB" -tAc "INSERT INTO projects (organization_id, name) VALUES ('$ORG', 'smoke-proj-${RANDOM}') RETURNING id;" | head -1)

# --- Create service token (via psql — dashboard routes require vault unlock) ---

ze_TOKEN="ze_development_$(python3 -c "import secrets;print(secrets.token_hex(32))")"
ze_HASH=$(python3 -c "import hashlib;print(hashlib.sha256('$ze_TOKEN'.encode()).hexdigest())")
psql "$DB" -c "INSERT INTO service_tokens (project_id, name, environment, token_hash, permission) VALUES ('$PID', 'smoke', 'development', decode('$ze_HASH', 'hex'), 'read_write');" >/dev/null

# --- SDK: Secrets CRUD ---

NH=$(b64rand 32)
CT=$(b64rand 64)
NC=$(b64rand 12)

echo ""
echo "=== SDK: Create Secret ==="
CREATE_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$API/v1/sdk/secrets" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ze_TOKEN" \
  -d '{"project_id":"'"$PID"'","environment":"development","name_hash":"'"$NH"'","ciphertext":"'"$CT"'","nonce":"'"$NC"'"}')
assert_status "POST /v1/sdk/secrets" "201" "$CREATE_STATUS"

echo ""
echo "=== SDK: Create Duplicate ==="
DUP_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$API/v1/sdk/secrets" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ze_TOKEN" \
  -d '{"project_id":"'"$PID"'","environment":"development","name_hash":"'"$NH"'","ciphertext":"'"$CT"'","nonce":"'"$NC"'"}')
assert_status "POST /v1/sdk/secrets (duplicate)" "409" "$DUP_STATUS"

echo ""
echo "=== SDK: List Secrets ==="
LIST_RESP=$(curl -s -w "\n%{http_code}" "$API/v1/sdk/secrets?project_id=$PID&environment=development" \
  -H "Authorization: Bearer $ze_TOKEN")
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
  -H "Authorization: Bearer $ze_TOKEN" \
  -d '{"ciphertext":"'"$CT2"'","nonce":"'"$NC2"'"}')
UPDATE_STATUS=$(echo "$UPDATE_RESP" | tail -1)
UPDATE_BODY=$(echo "$UPDATE_RESP" | sed '$d')
assert_status "PUT /v1/sdk/secrets/:nameHash" "200" "$UPDATE_STATUS"
assert_contains "version bumped" "$UPDATE_BODY" '"version":2'

echo ""
echo "=== SDK: Delete Secret ==="
DEL_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$API/v1/sdk/secrets/$NH_URL?project_id=$PID&environment=development" \
  -H "Authorization: Bearer $ze_TOKEN")
assert_status "DELETE /v1/sdk/secrets/:nameHash" "200" "$DEL_STATUS"

echo ""
echo "=== SDK: Get Deleted (should 404) ==="
GONE_STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$API/v1/sdk/secrets/$NH_URL?project_id=$PID&environment=development" \
  -H "Authorization: Bearer $ze_TOKEN")
assert_status "GET deleted secret" "404" "$GONE_STATUS"

# --- SDK: Auth checks ---

echo ""
echo "=== SDK: Invalid Token ==="
BAD_STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$API/v1/sdk/secrets?project_id=$PID&environment=development" \
  -H "Authorization: Bearer ze_dev_fakefake")
assert_status "GET with invalid token" "401" "$BAD_STATUS"

# --- CLI tests ---

echo ""
echo "=== CLI: Setup ==="
export ZENV_TOKEN="$ze_TOKEN"
export ZENV_VAULT_KEY="smoke-test-vault-key"
export ZENV_PROJECT="$PID"
export ZENV_ENV="development"

echo ""
echo "=== CLI: whoami ==="
WHOAMI_OUT=$(./bin/zenv whoami 2>&1)
assert_contains "whoami shows API" "$WHOAMI_OUT" "API:"
assert_contains "whoami shows token" "$WHOAMI_OUT" "Token:"

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
