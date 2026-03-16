#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${IDEASAVER_BASE_URL:-http://127.0.0.1:8080}"
TOKEN="${IDEASAVER_TOKEN:-}"
SHARE_CODE="${IDEASAVER_SHARE_CODE:-}"

echo "[Smoke] BASE_URL=${BASE_URL}"

check_http_200() {
  local url="$1"
  local name="$2"
  local code
  code=$(curl -sS -o /dev/null -w "%{http_code}" "$url")
  if [[ "$code" != "200" ]]; then
    echo "[FAIL] ${name} -> HTTP ${code} (${url})"
    exit 1
  fi
  echo "[OK] ${name}"
}

check_service() {
  if command -v systemctl >/dev/null 2>&1; then
    if ! systemctl is-active --quiet ideasaver; then
      echo "[FAIL] ideasaver service is not active"
      exit 1
    fi
    echo "[OK] ideasaver service is active"
  else
    echo "[WARN] systemctl not found, skip service check"
  fi
}

check_service
check_http_200 "${BASE_URL}/api/system/forbidden-image" "forbidden image route"

login_resp="$(curl -sS "${BASE_URL}/api/auth/login")"
if [[ "${login_resp}" != *"\"url\""* ]]; then
  echo "[FAIL] login endpoint response invalid: ${login_resp}"
  exit 1
fi
echo "[OK] login endpoint"

if [[ -n "${TOKEN}" ]]; then
  code=$(curl -sS -o /dev/null -w "%{http_code}" \
    -H "Authorization: Bearer ${TOKEN}" \
    "${BASE_URL}/api/user/me")
  if [[ "$code" != "200" ]]; then
    echo "[FAIL] /api/user/me with token -> HTTP ${code}"
    exit 1
  fi
  echo "[OK] /api/user/me with token"
else
  echo "[WARN] IDEASAVER_TOKEN not set, skip authenticated checks"
fi

if [[ -n "${SHARE_CODE}" ]]; then
  code=$(curl -sS -o /dev/null -w "%{http_code}" \
    "${BASE_URL}/api/shares/${SHARE_CODE}")
  if [[ "$code" != "200" && "$code" != "403" ]]; then
    echo "[FAIL] share access check returned HTTP ${code}"
    exit 1
  fi
  echo "[OK] share access endpoint (${code})"
else
  echo "[WARN] IDEASAVER_SHARE_CODE not set, skip share endpoint check"
fi

echo "[PASS] smoke test completed successfully"
