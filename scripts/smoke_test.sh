#!/usr/bin/env bash
# Post-deploy smoke test for FITS Processor.
#
#   ADMIN_EMAIL=admin@fits.local ADMIN_PASSWORD='...' ./scripts/smoke_test.sh http://localhost:8080
#
# Needs: bash, curl, python3. Creates one throw-away user and deletes it at the end.
# It sends ~6 failed logins for a fake e-mail to test the brute-force limiter
# (set SMOKE_SKIP_THROTTLE=1 to skip that part).
set -u
BASE="${1:-http://localhost:8080}"
EMAIL="${ADMIN_EMAIL:-admin@fits.local}"
PASS="${ADMIN_PASSWORD:?set ADMIN_PASSWORD}"
H_CSRF=(-H "X-Requested-With: fits" -H "Content-Type: application/json")
TMP="$(mktemp -d)"; trap 'rm -rf "$TMP"' EXIT
FAILS=0

ok()   { printf '  \033[32mPASS\033[0m %s\n' "$1"; }
bad()  { printf '  \033[31mFAIL\033[0m %s\n' "$1"; FAILS=$((FAILS+1)); }
check(){ if [ "$2" = "$3" ]; then ok "$1"; else bad "$1 (got '$2', want '$3')"; fi; }
code() { curl -s -o /dev/null -w '%{http_code}' "$@"; }
jget() { python3 -c "import sys,json;d=json.load(sys.stdin);print(eval('d'+sys.argv[1]))" "$1" 2>/dev/null; }

echo "== health & headers"
check "GET /health" "$(code "$BASE/health")" 200
check "GET /ready (database reachable)" "$(code "$BASE/ready")" 200
HDRS="$(curl -sI "$BASE/health")"
echo "$HDRS" | grep -qi '^content-security-policy:' && ok "CSP header present" || bad "CSP header missing"
echo "$HDRS" | grep -qi '^x-content-type-options: nosniff' && ok "nosniff header present" || bad "nosniff missing"

echo "== authentication"
check "login without CSRF header is rejected" "$(code -X POST "$BASE/api/auth/login" -H 'Content-Type: application/json' -d '{"email":"a@b.c","password":"x"}')" 403
curl -s -D "$TMP/h" -c "$TMP/jar" -X POST "$BASE/api/auth/login" "${H_CSRF[@]}" -d "{\"email\":\"$EMAIL\",\"password\":\"$PASS\"}" > "$TMP/login.json"
AT="$(jget '["access_token"]' < "$TMP/login.json")"
[ -n "$AT" ] && ok "admin login" || { bad "admin login failed - check ADMIN_EMAIL / ADMIN_PASSWORD"; exit 1; }
if [ "$(jget '["user"]["must_change_password"]' < "$TMP/login.json")" = "True" ]; then
  echo "  This admin account still has a temporary password (must_change_password=true)."
  echo "  Log in once in the browser and choose a new password - or set ADMIN_PASSWORD in .env before the first start - then re-run."
  exit 2
fi
grep -qi 'set-cookie: fits_rt=.*HttpOnly' "$TMP/h" && ok "refresh cookie is HttpOnly" || bad "refresh cookie not HttpOnly"
grep -qi 'set-cookie: fits_rt=.*SameSite=Strict' "$TMP/h" && ok "refresh cookie is SameSite=Strict" || bad "refresh cookie not SameSite=Strict"
grep -q refresh_token "$TMP/login.json" && bad "refresh token leaked into JSON body" || ok "refresh token is not in the JSON body"
A=(-H "Authorization: Bearer $AT")
check "GET /api/users/me" "$(code "$BASE/api/users/me" "${A[@]}")" 200
check "no token -> 401" "$(code "$BASE/api/files")" 401
check "refresh via cookie" "$(code -b "$TMP/jar" -c "$TMP/jar2" -X POST "$BASE/api/auth/refresh" "${H_CSRF[@]}")" 200
check "replaying the OLD refresh cookie -> 401" "$(code -b "$TMP/jar" -X POST "$BASE/api/auth/refresh" "${H_CSRF[@]}")" 401
M1="$(curl -s -X POST "$BASE/api/auth/login" "${H_CSRF[@]}" -d '{"email":"ghost@nowhere.test","password":"wrong-pass-1"}')"
M2="$(curl -s -X POST "$BASE/api/auth/login" "${H_CSRF[@]}" -d "{\"email\":\"$EMAIL\",\"password\":\"wrong-pass-1\"}")"
check "same error for unknown user and wrong password" "$M1" "$M2"

echo "== scan path restriction"
check "scan_dir outside the allowed root -> 400" "$(code -X POST "$BASE/api/scan" "${A[@]}" "${H_CSRF[@]}" -d '{"scan_dir":"/etc"}')" 400
check "scan_dir with .. -> 400" "$(code -X POST "$BASE/api/scan" "${A[@]}" "${H_CSRF[@]}" -d '{"scan_dir":"../../etc"}')" 400

echo "== users, forced password change, sessions"
UEMAIL="smoke-$(date +%s)@test.local"; UPASS="Comet-Trail-4242"; NEWPASS="Nebula-Orbit-9911"
check "weak password rejected on create" "$(code -X POST "$BASE/api/users" "${A[@]}" "${H_CSRF[@]}" -d "{\"email\":\"$UEMAIL\",\"password\":\"12345678\",\"full_name\":\"Smoke\",\"role\":\"viewer\"}")" 400
check "create user" "$(code -X POST "$BASE/api/users" "${A[@]}" "${H_CSRF[@]}" -d "{\"email\":\"$UEMAIL\",\"password\":\"$UPASS\",\"full_name\":\"Smoke\",\"role\":\"viewer\"}")" 201
curl -s -c "$TMP/ujar" -X POST "$BASE/api/auth/login" "${H_CSRF[@]}" -d "{\"email\":\"$UEMAIL\",\"password\":\"$UPASS\"}" > "$TMP/ul.json"
UAT="$(jget '["access_token"]' < "$TMP/ul.json")"; UID_="$(jget '["user"]["id"]' < "$TMP/ul.json")"
UA=(-H "Authorization: Bearer $UAT")
check "new user is flagged must_change_password" "$(jget '["user"]["must_change_password"]' < "$TMP/ul.json")" True
check "other endpoints are blocked until the password is changed" "$(code "$BASE/api/files" "${UA[@]}")" 403
check "...but /api/users/me is allowed" "$(code "$BASE/api/users/me" "${UA[@]}")" 200
check "user changes password" "$(code -b "$TMP/ujar" -X PUT "$BASE/api/users/me/password" "${UA[@]}" "${H_CSRF[@]}" -d "{\"old_password\":\"$UPASS\",\"new_password\":\"$NEWPASS\"}")" 200
check "endpoints work after the change" "$(code "$BASE/api/files" "${UA[@]}")" 200
check "viewer cannot list users (403)" "$(code "$BASE/api/users" "${UA[@]}")" 403
check "viewer cannot start a scan (403)" "$(code -X POST "$BASE/api/scan" "${UA[@]}")" 403
SID="$(curl -s -b "$TMP/ujar" "$BASE/api/auth/sessions" "${UA[@]}" | jget '["data"][0]["id"]')"
[ -n "$SID" ] && ok "sessions list works" || bad "sessions list empty"
check "revoke own session" "$(code -X DELETE "$BASE/api/auth/sessions/$SID" "${UA[@]}" "${H_CSRF[@]}")" 204
check "admin deactivates the user" "$(code -X PUT "$BASE/api/users/$UID_" "${A[@]}" "${H_CSRF[@]}" -d '{"full_name":"Smoke","role":"viewer","is_active":false}')" 200
check "deactivated user's token stops working immediately" "$(code "$BASE/api/files" "${UA[@]}")" 401
check "admin cannot deactivate themself" "$(code -X PUT "$BASE/api/users/$(curl -s "$BASE/api/users/me" "${A[@]}" | jget '["data"]["id"]')" "${A[@]}" "${H_CSRF[@]}" -d '{"full_name":"x","role":"admin","is_active":false}')" 400
check "delete the smoke user" "$(code -X DELETE "$BASE/api/users/$UID_" "${A[@]}")" 204

echo "== audit log"
N="$(curl -s "$BASE/api/audit-logs?page_size=5" "${A[@]}" | jget '["total"]')"
[ "${N:-0}" -gt 0 ] 2>/dev/null && ok "audit log has entries ($N)" || bad "audit log is empty"

if [ -z "${SMOKE_SKIP_THROTTLE:-}" ]; then
  echo "== brute-force limiter"
  LAST=0; for _ in 1 2 3 4 5 6 7; do LAST="$(code -X POST "$BASE/api/auth/login" "${H_CSRF[@]}" -d '{"email":"victim@nowhere.test","password":"wrong-pass-1"}')"; done
  check "7th failed attempt is throttled (429)" "$LAST" 429
fi

echo
if [ "$FAILS" -eq 0 ]; then printf '\033[32mAll checks passed.\033[0m\n'; else printf '\033[31m%d check(s) failed.\033[0m\n' "$FAILS"; exit 1; fi
