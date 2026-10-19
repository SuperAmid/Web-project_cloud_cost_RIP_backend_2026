# Макет фронтенда provider_routers

SVG-экспорты экранов, к которым обращается веб-сервис ЛР3:

| Файл | Страница | Домен API |
| --- | --- | --- |
| `01-feed.svg` | Лента (карточка, лайк, «следующий») | `/api/provider_routers/feed`, `/api/provider_routers/{id}/like` |
| `02-draft.svg` | Добавление (шаг 1 — название, фото, видео; шаг 2 — публикация) | `POST /api/provider_routers`, `GET …/draft`, `PUT …/{id}/publish` |
| `03-routers.svg` | Маршрутизаторы (плитка с фильтром, «Удалить») | `GET /api/provider_routers`, `DELETE /api/provider_routers/{id}` |

Четвёртая страница — «Вход и регистрация» (`/api/provider_router_users/*`). Макета у неё нет:
в ЛР4 вход выдаёт JWT, выход заносит токен в blacklist Redis, а проверяются эти методы через Swagger
(`/swagger/`) и Postman. На диаграмме классов (`docs/provider_routers.png`) страница зависит от домена пользователя.

Исходный макет — файл Figma (ссылка прикладывается к отчёту); здесь лежат только экспорты.
