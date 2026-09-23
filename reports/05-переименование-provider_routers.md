# Отчёт 05 — переименование всего в `provider_routers`

Замечание: «ВСЕ ПЕРЕДЕЛАТЬ: шаблоны тоже, ВСЕ ФАЙЛЫ и т. д. — код, URL и т. д.
переименовать в `provider_routers`».

## Было

- URL-маршруты: `/provider-routers/...` (дефис) в `main.go`, в ленте и плитке;
- ссылки навигации и плиток: `/routers/...` в `partials.html` и `grid.html`;
- модуль Go: `provider-router-rip`;
- README-ссылки: `http://localhost:8080/routers/...`.

Итог: в коде и шаблонах использовались два разных формата — дефисные пути
и `/routers`, без единого нейминга.

## Стало

Единый нейминг `provider_routers` (без дефисов) во **всех** файлах:

| Файл | Было | Стало |
| ---- | ---- | ----- |
| `main.go` | `/provider-routers/feed`, `/provider-routers/draft`, `/provider-routers`, лог `provider-routers` | `/provider_routers/...`, лог `provider_routers` |
| `templates/feed.html` | `/provider-routers/feed?id=...` | `/provider_routers/feed?id=...` |
| `templates/grid.html` | action `/provider-routers`, тайлы `/routers/feed?id=...` | `/provider_routers` и `/provider_routers/feed?id=...` |
| `templates/partials.html` | `/routers/feed`, `/routers/draft`, `/routers` | `/provider_routers/...` |
| `templates/draft.html` | без ссылок | без изменений |
| `go.mod` | `module provider-router-rip` | `module provider_routers` |
| `README.md` | `http://localhost:8080/routers/...` | `http://localhost:8080/provider_routers/...` |
| `reports/01–04` | `/provider-routers/...` | `/provider_routers/...` |

## Проверка

`grep 'provider-routers\|/routers' -r` по проекту (кроме `reports/00` и `reports/05`,
где оба варианта описаны намеренно) не даёт совпадений. Все GET-маршруты и ссылки
используют только `provider_routers`.