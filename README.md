# provider_routers — ЛР‑3

Лабораторная работа №3: веб-сервис (REST API) для темы «Маршрутизаторы сети провайдера». Шаблонов и HTML больше нет — сервис отдаёт только JSON, проверяется через Postman/Insomnia. Данные в PostgreSQL (через GORM), фото и видео — в MinIO.

Ветки: `ssr-design` — ЛР1, `database-orm` — ЛР2, `web-service` — ЛР3.

## Запуск

```powershell
docker compose up -d
go run .
```

API: `http://localhost:8080/api`. PostgreSQL: `localhost:5433`, БД `provider_routers`, пользователь `provider_routers_user`, пароль `provider_routers_password`. Adminer: `http://localhost:8081`. MinIO Console: `http://localhost:9001` (`minioadmin` / `minioadmin`), бакет `provider-media`.

Таблицы создаются/обновляются через GORM `AutoMigrate` при старте; пустая БД заполняется начальными данными. SQL-миграции для ручного применения лежат в `migrations/` (`001_init`, `002_provider_router_users_password`), начальные данные — в `db/seed.sql`.

Коллекция запросов: `docs/postman/provider_routers_lab3.postman_collection.json` (импортируется и в Postman, и в Insomnia).

```powershell
go test ./...
```

## Структура кода

| Папка | Что внутри |
| --- | --- |
| `main.go` | конфиг → MinIO → БД → миграции → HTTP-сервер |
| `internal/config` | настройки из переменных окружения |
| `internal/ds` | модели GORM = таблицы БД |
| `internal/serializer` | структуры запросов и ответов JSON (сериализация/десериализация) |
| `internal/repository` | бизнес-логика и работа с БД через ORM, MinIO, функция-singleton текущего пользователя |
| `internal/handler` | HTTP-обработчики: разбор запроса, проверка полей, коды ответа |
| `internal/api` | маршрутизация URL → обработчик |

**Пользователь без авторизации.** В ЛР3 создатель зафиксирован константой `fixedProviderRouterUserEmail` в `internal/repository/current_user.go`. Функция-singleton `CurrentProviderRouterUser()` один раз находит его в БД, запоминает и дальше возвращает тот же объект; её вызывают все методы домена маршрутизатора.

**Системные поля** (`id`, `status`, `created_by_id`, `created_at`, `published_at`) с клиента не принимаются: в структурах запросов их нет, а JSON читается с `DisallowUnknownFields`, поэтому попытка их передать — ошибка `400`.

**Статусы:** `draft → published` (метод публикации), `draft | published → deleted` (метод удаления). Вернуть запись в `draft` нельзя. Записи `deleted` клиенту не отдаются.

## HTTP-методы

Все ответы — JSON. Ошибка всегда `{"error": "текст"}`.

### Домен маршрутизатора `/api/provider_routers`

| # | Метод и URL | Что делает | Ответ |
| --- | --- | --- | --- |
| 1 | `GET /api/provider_routers?min_bandwidth_mbps=10000&max_cost_rub=500000` | Плитка: только `published`, фильтры необязательные и выполняются в БД. `is_own` = 1, если создатель — текущий пользователь | `200` массив |
| 2 | `GET /api/provider_routers/feed` | Лента: первая опубликованная карточка. `is_liked` = 1, если текущий пользователь лайкнул | `200` / `404` |
| 2 | `GET /api/provider_routers/feed?id=102` | Карточка ленты по id | `200` / `404` |
| 2 | `GET /api/provider_routers/feed?id=102&next=true` | Следующая после id (после последней — снова первая). БД возвращает одну строку | `200` / `404` |
| 3 | `GET /api/provider_routers/draft` | Черновик текущего пользователя, id не указывается | `200` / `404` |
| 4 | `POST /api/provider_routers` | Создание черновика. `multipart/form-data`: `name` (текст), `image` (файл .jpg/.jpeg/.png/.webp/.svg до 10 МБ), `video` (файл .mp4/.webm/.mov до 50 МБ). Файлы сохраняются в MinIO под сгенерированными латинскими именами `provider_routers/router-<дата>-<hex>.png`, `videos/router-<дата>-<hex>.mp4`, имена — в `image_key`, `video_key` | `201` / `409`, если черновик уже есть |
| 5 | `PUT /api/provider_routers/{id}/publish` | Публикация своего черновика: `draft → published`, `published_at` ставит сервер | `200` / `403` / `404` / `409` |
| 6 | `DELETE /api/provider_routers/{id}` | Логическое удаление своей записи: `status = deleted` | `200` / `403` / `404` |
| 7 | `POST /api/provider_routers/{id}/like` | Лайк текущего пользователя: `{"like": 1}` ставит, `{"like": 0}` снимает | `200` / `404` |

Тело публикации (обязательны `description`, `bandwidth_mbps` > 0, `cost_rub` > 0; `router_type` — `central`, `intermediate` или `residential`, по умолчанию `residential`):

```json
{
  "description": "Конечный маршрутизатор ЖК «Озерки» на 240 квартир.",
  "router_type": "residential",
  "bandwidth_mbps": 2500,
  "cost_rub": 84000
}
```

Элемент плитки:

```json
{
  "id": 101, "name": "Core Backbone One", "description": "…",
  "router_type": "central", "bandwidth_mbps": 100000, "cost_rub": 980000,
  "image_url": "http://localhost:9000/provider-media/provider_routers/core.svg",
  "video_url": "http://localhost:9000/provider-media/videos/core-loop.mp4",
  "likes_count": 2, "is_own": 0, "published_at": "2026-10-17T15:13:10+03:00"
}
```

Карточка ленты — те же поля, но вместо `is_own` — `is_liked` и добавлено `creator_name`. Черновик и ответы создания/публикации/удаления содержат `status`, `image_key`, `video_key`, `created_at`.

### Домен пользователя `/api/provider_router_users`

| # | Метод и URL | Что делает | Ответ |
| --- | --- | --- | --- |
| 8 | `POST /api/provider_router_users/register` | Регистрация: `email`, `display_name`, `password` (6–72 байта). Пароль хранится как bcrypt-хеш | `201` / `409`, если email занят |
| 9 | `POST /api/provider_router_users/login` | Заглушка до ЛР4: проверяет `email` + `password`, сессию/JWT не выдаёт | `200` / `401` |
| 10 | `POST /api/provider_router_users/logout` | Заглушка до ЛР4 | `200` |

## Таблицы БД

Каскадного удаления нет: все внешние ключи `ON UPDATE RESTRICT ON DELETE RESTRICT`.

### `provider_router_users` — пользователи

| Поле | Тип | Описание |
| --- | --- | --- |
| `id` | `BIGSERIAL` PK | |
| `email` | `VARCHAR(120)` UNIQUE NOT NULL | логин |
| `display_name` | `VARCHAR(80)` NOT NULL | имя |
| `password_hash` | `VARCHAR(72)` NOT NULL DEFAULT `''` | bcrypt-хеш пароля (добавлен в ЛР3) |
| `created_at` | `TIMESTAMPTZ` NOT NULL | дата регистрации |

### `provider_routers` — маршрутизаторы (услуги)

| Поле | Тип | Описание |
| --- | --- | --- |
| `id` | `BIGSERIAL` PK | |
| `name` | `VARCHAR(120)` NOT NULL | наименование |
| `description` | `VARCHAR(500)` NOT NULL | краткое описание |
| `status` | `VARCHAR(12)` NOT NULL | `draft` / `published` / `deleted` |
| `image_key` | `VARCHAR(160)` NOT NULL | ключ изображения в MinIO |
| `video_key` | `VARCHAR(160)` NOT NULL | ключ видео в MinIO |
| `router_type` | `VARCHAR(24)` NOT NULL | `central` / `intermediate` / `residential` |
| `bandwidth_mbps` | `INTEGER` NOT NULL | **поле по теме:** пропускная способность, Мбит/с |
| `cost_rub` | `INTEGER` NOT NULL | **поле по теме:** стоимость, ₽ |
| `created_at` | `TIMESTAMPTZ` NOT NULL | дата создания |
| `published_at` | `TIMESTAMPTZ` NULL | дата формирования (публикации) |
| `created_by_id` | `BIGINT` FK → `provider_router_users.id` | создатель |

Частичный уникальный индекс `one_draft_provider_router_per_creator (created_by_id) WHERE status = 'draft'` — не больше одного черновика у пользователя.

### `provider_router_likes` — лайки (м-м пользователь–маршрутизатор)

| Поле | Тип | Описание |
| --- | --- | --- |
| `provider_router_id` | `BIGINT` PK, FK → `provider_routers.id` | |
| `user_id` | `BIGINT` PK, FK → `provider_router_users.id` | |
| `created_at` | `TIMESTAMPTZ` NOT NULL | когда поставлен лайк |

## Порядок показа (Postman/Insomnia, браузер не нужен)

1. Импортировать коллекцию, во втором запросе выбрать файлы для `image` и `video` (например, `assets/provider-media/provider_routers/core.svg` и `assets/provider-media/videos/core-loop.mp4`).
2. Если у студента уже есть черновик, сначала удалить его (`DELETE`) или опубликовать — иначе создание вернёт `409`.
3. Выполнить запросы по порядку: список с фильтром → добавление с картинкой и видео → черновик → публикация → лента без id → лента `?id=…&next=true` → лайк → удаление → регистрация (→ вход/выход-заглушки). Скрипты в коллекции сами сохраняют переменные: id созданного черновика → `provider_router_id` (публикация, лайк, удаление), id из «ленты без id» → `feed_id` (запрос `next=true`), email из регистрации → `user_email` (вход). Email регистрации уникален за счёт `{{$timestamp}}`, поэтому запрос можно повторять.
4. Изменения в БД (Adminer → SQL-запрос):

```sql
SELECT id, name, status, image_key, video_key, bandwidth_mbps, cost_rub, created_at, published_at, created_by_id
FROM provider_routers ORDER BY id;
SELECT * FROM provider_router_likes ORDER BY provider_router_id, user_id;
SELECT id, email, display_name, left(password_hash, 7) AS hash_prefix FROM provider_router_users ORDER BY id;
```

Загруженные файлы видны в MinIO Console в бакете `provider-media`.
