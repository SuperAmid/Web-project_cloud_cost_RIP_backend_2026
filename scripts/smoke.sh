#!/usr/bin/env bash
# Smoke-тест всего стека ЛР3: реальные PostgreSQL + MinIO + API.
# Сценарий: создание (с загрузкой фото и видео в MinIO) → публикация → лайк → удаление.
# Падает с кодом 1, если стек не поднялся или любой шаг вернул не тот код.
#   docker compose up -d && go run .   # в другом окне
#   make smoke                         # или bash scripts/smoke.sh
set -euo pipefail

API="${API:-http://localhost:8080/api}"
MINIO="${MINIO:-http://localhost:9000}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TMP="$(mktemp -d)"; trap 'rm -rf "$TMP"' EXIT

fail() { echo "SMOKE FAIL: $*" >&2; exit 1; }
step() { echo "== $*"; }
# expect <код> <метод> <url> [аргументы curl…] — печатает тело ответа в $TMP/body
expect() {
  local want="$1" method="$2" url="$3"; shift 3
  local got
  got="$(curl -sS -o "$TMP/body" -w '%{http_code}' -X "$method" "$@" "$url")" || fail "$method $url: нет соединения"
  [ "$got" = "$want" ] || fail "$method $url: код $got, ожидался $want: $(cat "$TMP/body")"
}
json() { python3 -c "import json,sys; print(json.load(open('$TMP/body'))$1)"; }

step "MinIO живой и бакет заполнен (minio-init отработал)"
expect 200 GET "$MINIO/minio/health/live"
expect 200 GET "$MINIO/provider-media/provider_routers/core.svg"

step "список с фильтром"
expect 200 GET "$API/provider_routers?min_bandwidth_mbps=1000"
[ "$(json '.__len__()')" -ge 1 ] || fail "список пуст — сид не выполнен"
image_url="$(json '[0]["image_url"]')"
expect 200 GET "$image_url"

step "удаление старого черновика студента, если остался от прошлого запуска"
if [ "$(curl -s -o "$TMP/body" -w '%{http_code}' "$API/provider_routers/draft")" = 200 ]; then
  expect 200 DELETE "$API/provider_routers/$(json '["id"]')"
fi

step "создание с фото и видео"
expect 201 POST "$API/provider_routers" \
  -F "name=Smoke Router $(date +%s)" \
  -F "image=@$ROOT/assets/provider-media/provider_routers/core.svg" \
  -F "video=@$ROOT/assets/provider-media/videos/core-loop.mp4"
id="$(json '["id"]')"
uploaded="$(json '["image_url"]')"
expect 200 GET "$uploaded"   # файл реально лежит в MinIO
expect 200 GET "$API/provider_routers/draft"

step "публикация #$id"
expect 200 PUT "$API/provider_routers/$id/publish" -H 'Content-Type: application/json' \
  -d '{"description":"smoke-тест","bandwidth_mbps":2500,"cost_rub":84000}'
[ "$(json '["status"]')" = published ] || fail "статус после публикации не published"
expect 409 PUT "$API/provider_routers/$id/publish" -H 'Content-Type: application/json' \
  -d '{"description":"повтор","bandwidth_mbps":1,"cost_rub":1}'

step "лента и лайк"
expect 200 GET "$API/provider_routers/feed?id=$id"
expect 200 POST "$API/provider_routers/$id/like" -H 'Content-Type: application/json' -d '{"like":1}'
[ "$(json '["is_liked"]')" = 1 ] || fail "лайк не поставлен"
expect 200 GET "$API/provider_routers/feed?id=$id&next=true"

step "удаление"
expect 200 DELETE "$API/provider_routers/$id"
expect 404 GET "$API/provider_routers/feed?id=$id"

step "регистрация"
expect 201 POST "$API/provider_router_users/register" -H 'Content-Type: application/json' \
  -d "{\"email\":\"smoke$(date +%s%N)@provider-routers.local\",\"display_name\":\"Smoke\",\"password\":\"secret1\"}"

echo "SMOKE OK"
