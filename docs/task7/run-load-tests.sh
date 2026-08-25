#!/usr/bin/env bash
# Нагрузочное тестирование ручек диалогов (internal/api/dialog.go) утилитой hey.
#
# Требуется:
#   TOKEN — JWT пользователя-источника (см. POST /api/v1/login)
#   DST   — user_id собеседника
#
# Каждый прогон длится 30 секунд, таймаут запроса — значение hey по умолчанию (20с).
set -u

BASE="${BASE:-http://localhost:8282/api/v1}"
TOKEN="${TOKEN:?TOKEN is required}"
DST="${DST:?DST is required}"
OUT="$(cd "$(dirname "$0")" && pwd)/raw"
DURATION="${DURATION:-30s}"

mkdir -p "$OUT"

run() {
  local name="$1" conc="$2"; shift 2
  echo ">>> ${name} c=${conc}"
  {
    echo "# route:       ${name}"
    echo "# concurrency: ${conc}"
    echo "# duration:    ${DURATION}"
    echo "# command:     hey -z ${DURATION} -c ${conc} $*"
    echo
    hey -z "$DURATION" -c "$conc" "$@"
  } >"${OUT}/${name}_c${conc}.log" 2>&1
}

for C in 1 10 100 1000; do
  run dialog_list "$C" \
    -H "Authorization: Bearer ${TOKEN}" \
    "${BASE}/dialog/list"
done

for C in 1 10 100 1000; do
  run dialog_messages "$C" \
    -H "Authorization: Bearer ${TOKEN}" \
    "${BASE}/dialog/${DST}/list?limit=100&offset=0"
done

for C in 1 10 100 1000; do
  run dialog_send "$C" -m POST \
    -H "Authorization: Bearer ${TOKEN}" -T 'application/json' \
    -d '{"text":"load test message"}' \
    "${BASE}/dialog/${DST}/send"
done

echo "done -> ${OUT}"
