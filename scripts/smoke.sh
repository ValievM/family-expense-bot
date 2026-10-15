#!/usr/bin/env bash
# Сквозная проверка собранного стека: API через nginx, создание, фильтр, изменение, удаление.
# Использование: docker compose up -d --build --wait && scripts/smoke.sh
set -euo pipefail

base="${BASE_URL:-http://localhost:${WEB_PORT:-8080}}"
json=(-H 'Content-Type: application/json')

# Принимает: сообщение об ошибке. Делает: печатает его в stderr и завершает скрипт. Возвращает: код 1.
fail() { echo "SMOKE FAILED: $1" >&2; exit 1; }

# Принимает: ожидаемый код, метод, путь, необязательное тело. Делает: вызывает API и сверяет код ответа. Возвращает: тело ответа в stdout.
call() {
  local want="$1" method="$2" path="$3" body="${4:-}" out code
  out="$(mktemp)"
  if [[ -n "$body" ]]; then
    code="$(curl -s -o "$out" -w '%{http_code}' -X "$method" "${json[@]}" -d "$body" "$base$path")"
  else
    code="$(curl -s -o "$out" -w '%{http_code}' -X "$method" "$base$path")"
  fi
  [[ "$code" == "$want" ]] || fail "$method $path returned $code, want $want: $(cat "$out")"
  cat "$out"
  rm -f "$out"
}

call 200 GET /healthz >/dev/null
call 200 GET / | grep -q '<div id="root"' || fail "web panel index is not served"

created="$(call 201 POST /api/expenses '{"user_id":900001,"amount":35000,"category":"food","comment":"smoke coffee"}')"
id="$(sed -E 's/.*"id":([0-9]+).*/\1/' <<<"$created")"
call 201 POST /api/expenses '{"user_id":900002,"amount":120000,"category":"transport","spent_at":"2026-01-15"}' >/dev/null

only_first="$(call 200 GET '/api/expenses?user_id=900001')"
grep -q 'smoke coffee' <<<"$only_first" || fail "filter by user lost the expense"
! grep -q '"user_id":900002' <<<"$only_first" || fail "filter by user leaked another member"

call 200 PATCH "/api/expenses/$id" '{"amount":45000}' | grep -q '"amount":45000' || fail "patch did not change the amount"
call 400 POST /api/expenses '{"user_id":900001,"amount":0,"category":"food"}' >/dev/null
call 404 DELETE /api/expenses/999999999 >/dev/null
call 204 DELETE "/api/expenses/$id" >/dev/null

echo "SMOKE OK"
