# provider_routers — ЛР‑4

Лабораторная работа №4: авторизация и документация веб-сервиса темы «Маршрутизаторы сети провайдера». Вход выдаёт JWT, выход заносит токен в blacklist Redis; все методы описаны в Swagger. Данные в PostgreSQL (через GORM), фото и видео — в MinIO.

Ветки: `ssr-design` — ЛР1, `database-orm` — ЛР2, `web-service` — ЛР3, `auth-swagger` — ЛР4.

## Запуск

```powershell
docker compose up -d     # PostgreSQL, Adminer, MinIO, Redis; minio-init загружает картинки в бакет
docker compose ps        # minio, postgres, redis — healthy, minio-init — Exited (0)
go run .                 # API на http://localhost:8080/api, Swagger на http://localhost:8080/swagger/
```

API: `http://localhost:8080/api`. PostgreSQL: `localhost:5433`, БД `provider_routers`, пользователь `provider_routers_user`, пароль `provider_routers_password`. Adminer: `http://localhost:8081`. MinIO Console: `http://localhost:9001` (`minioadmin` / `minioadmin`), бакет `provider-media`. Redis: `localhost:6379`, пароль `provider_routers_redis_password`. Swagger UI: `http://localhost:8080/swagger/`, описание — `http://localhost:8080/swagger/doc.json`.

Демо-пользователи из начального наполнения: `student@provider-routers.local`, `operator@provider-routers.local`, `viewer@provider-routers.local`, пароль у всех — `password123`.

**Миграции.** При старте сервер выполняет версионированные SQL-файлы `migrations/*.up.sql` по порядку и записывает выполненные версии в таблицу `schema_migrations` (`AutoMigrate` больше не используется):

| Версия | Что делает |
| --- | --- |
| `001_init` | таблицы `provider_router_users`, `provider_routers`, `provider_router_likes`, индекс «один черновик» |
| `002_lab2_fixes` | исправления ЛР2: удалён `router_type`; `description`, `bandwidth_mbps`, `cost_rub` — NULL у черновика; у лайков свой `id`; убраны лишние `created_at` |
| `003_provider_router_users_password` | ЛР3: `password_hash` для регистрации |
| `004_demo_user_passwords` | ЛР4: пароль `password123` у трёх демо-пользователей, чтобы под ними можно было войти |

Если таблицы уже были созданы старой версией, `001_init` считается выполненной, остальные применяются поверх. Пустая БД заполняется начальными данными (то же — в `db/seed.sql` для ручного импорта).

Коллекция запросов: `docs/postman/provider_routers_lab4.postman_collection.json` (импортируется и в Postman, и в Insomnia). Диаграммы в одном файле StarUML `docs/provider_routers.mdj`: классов (`docs/provider_routers.png`), последовательности бизнес-процесса создателя: вход, список, добавление, черновик, публикация, лента, лайк, выход (`docs/provider_routers-sequence.png`), ER (`docs/provider_routers-er.png`).

## Тесты

| Команда | Что проверяет | Нужен Docker |
| --- | --- | --- |
| `go test ./...` | выдача и проверка JWT, чужой секрет, истёкший токен (`internal/auth`); разбор запросов (`internal/handler`); все 10 маршрутов с токеном, `401` у гостя на защищённых методах, вход → выход → токен в blacklist → `401`, Swagger (`internal/api`); расширения и размеры файлов, латинские имена (`internal/repository`) | нет |
| `TEST_DATABASE_URL=… go test ./internal/repository/` | переходы статусов на реальной PostgreSQL: `draft → published`, `published → deleted`, повторная публикация и возврат из `deleted` — ошибка, второй черновик — `409`, чужая запись — `403`, фильтр в БД, `next` после последней | да |

Интеграционным тестам нужна отдельная база в PostgreSQL из `docker compose`:

```powershell
docker compose exec postgres psql -U provider_routers_user -d provider_routers -c "CREATE DATABASE provider_routers_test"
$env:TEST_DATABASE_URL = "postgres://provider_routers_user:provider_routers_password@localhost:5433/provider_routers_test?sslmode=disable"
go test -count=1 ./internal/repository/
```

Без `TEST_DATABASE_URL` эти тесты пропускаются.

## Окружение

**Образы.** Официальные `quay.io/minio/minio` и `quay.io/minio/mc` стали приватными (анонимный токен без права `pull`), `minio/minio` удалён из Docker Hub. Поэтому MinIO берётся из архивных образов Bitnami — `bitnamilegacy/minio` и `bitnamilegacy/minio-client`: доступны анонимно, но обновляться не будут (для лабораторной подходит). Оба закреплены тегом версии и `@sha256` digest, чтобы стек не ломался от смены `latest`. У Bitnami нет аргумента `server /data`: порты задаются переменными `MINIO_API_PORT_NUMBER`/`MINIO_CONSOLE_PORT_NUMBER`, данные — в томе `/bitnami/minio/data`.

**Готовность.** У `minio` есть `healthcheck` (запрос `/minio/health/live`; в образе нет `curl`, поэтому через `/dev/tcp`), `minio-init` ждёт `condition: service_healthy` — порядок гарантирует compose, а не цикл в скрипте.

**Реестры и зеркала.** На машине проверки было два аплинка одновременно (Wi-Fi и PPPoE), поэтому маршрут до внешних реестров нестабилен — это внешний фактор, но его нужно учитывать при диагностике. Настройки Docker держать в `/etc/docker/daemon.json` (на Windows — Docker Desktop → Settings → Docker Engine), а не в `/etc/hosts`. Пример — `docs/docker/daemon.json`:

| Параметр | Значение | Зачем |
| --- | --- | --- |
| `registry-mirrors` | `https://mirror.gcr.io` | образы Docker Hub (`postgres`, `adminer`, `bitnamilegacy/*`) идут через зеркало |
| `dns` | резолверы, которые отвечают из вашей сети (в примере — роутер `192.168.50.1` и DNS PPPoE-провайдера `195.19.33.199`) | на машине проверки `8.8.8.8` и `1.1.1.1` не отвечали (`dig @8.8.8.8 quay.io` → timeout) — без рабочего DNS контейнеры молча не резолвят имена. Проверить: `dig +short @<адрес> quay.io` |
| `mtu` | `1400` | у PPPoE-аплинка MTU меньше 1500 (заголовок PPPoE + туннели); если сеть контейнеров отправляет пакеты по 1500 байт, крупные TLS-пакеты теряются и `pull` висит на `TLS handshake timeout`. На машине с обычным Ethernet/Wi-Fi параметр можно не указывать |

Строку вида `54.209.135.132 registry-1.docker.io` в `/etc/hosts` нужно удалить: адрес CDN меняется, «прибитый» адрес мёртв, и все прямые обращения к Docker Hub зависают. После правки проверить `docker pull postgres:16-alpine`.

**Если `pull` не проходит** — сначала понять класс ошибки, а не повторять `pull`:

| В ошибке | Значит | Что делать |
| --- | --- | --- |
| `401`, `403`, `unauthorized`, `no such manifest` при живой сети | образ стал приватным или нужен логин | сменить источник образа, `docker login` |
| `404`, `not found` | образ или тег удалён | другой тег/реестр |
| `timeout`, `handshake`, `aborted` | сеть | DNS, зеркала, MTU, маршрут |

```bash
docker manifest inspect bitnamilegacy/minio:2025.7.23-debian-12-r5   # есть ли образ вообще
curl -sSI https://quay.io/v2/minio/minio/manifests/latest | head -1   # 401 = доступ, таймаут = сеть
```

## Структура кода

| Папка | Что внутри |
| --- | --- |
| `main.go` | общие аннотации Swagger; конфиг → MinIO → БД → миграции → Redis → JWT → HTTP-сервер |
| `internal/config` | настройки из переменных окружения |
| `internal/auth` | `Manager`: выдача (`Issue`) и проверка (`Parse`) JWT, HS256 |
| `internal/redis` | клиент Redis: `WriteJWTToBlacklist`, `IsJWTBlacklisted` |
| `internal/ds` | модели GORM = таблицы БД, связи `Likes`, `CreatedBy` |
| `internal/serializer` | структуры запросов и ответов JSON (сериализация/десериализация) |
| `internal/repository` | бизнес-логика и работа с БД через ORM (`Preload`, `Association`, `OnConflict` — без SQL-подзапросов), миграции, MinIO за интерфейсом `MediaStorage` |
| `internal/handler` | HTTP-обработчики с аннотациями Swagger: разбор запроса, проверка полей, коды ответа; `auth.go` — middleware `Authenticate` и `RequireUser`. Зависят от интерфейсов `ProviderRouterRepository`, `ProviderRouterUserRepository`, `TokenBlacklist` (`interfaces.go`) |
| `internal/api` | маршрутизация URL → обработчик, раздача Swagger UI (`swagger.go`) |
| `migrations/` | версионированные SQL-миграции |
| `docs/swagger` | `swagger.json`, `swagger.yaml`, сгенерированные `swag init` |
| `docs/` | коллекция Postman, диаграммы (StarUML + PNG), макеты, пример `daemon.json` |

**Интерфейсы доменов.** Домен `/api/provider_routers` → интерфейс `ProviderRouterRepository`, домен `/api/provider_router_users` → `ProviderRouterUserRepository`, хранилище файлов → `MediaStorage`. Реализации — `*repository.Repository` и `*repository.Media`; соответствие проверяется при компиляции (`var _ Interface = (*Impl)(nil)`), в тестах подставляется подменный репозиторий.

**Авторизация.** `POST /api/provider_router_users/login` проверяет пароль и выдаёт JWT (HS256, `iss = provider_routers`, в claims — `user_id`, `email`, срок — `JWT_TTL`, по умолчанию 1 ч). Клиент передаёт его в заголовке `Authorization: Bearer <токен>`.

- `Authenticate` (обёртка над всеми маршрутами) разбирает токен и проверяет, нет ли его в blacklist Redis. Действительный токен кладёт пользователя в контекст запроса; без токена, с неверным или отозванным запрос идёт дальше как гость.
- `RequireUser` стоит на методах создателя (черновик, создание, публикация, удаление, лайк, выход) и отвечает гостю `401` с заголовком `WWW-Authenticate: Bearer`.
- Создатель записи и `is_own`/`is_liked` берутся из токена — `created_by_id` с клиента не передаётся. Singleton фиксированного пользователя из ЛР3 удалён.
- `POST /api/provider_router_users/logout` записывает токен в Redis: ключ `provider_routers:jwt:blacklist:<токен>`, значение — email, TTL — оставшийся срок жизни токена. После истечения JWT запись удаляется сама.

| Переменная | По умолчанию |
| --- | --- |
| `REDIS_ADDR` | `localhost:6379` |
| `REDIS_PASSWORD` | `provider_routers_redis_password` |
| `JWT_SECRET` | `provider_routers_dev_jwt_secret` |
| `JWT_TTL` | `1h` |

**Swagger.** Описание генерируется из аннотаций в `main.go` и `internal/handler` утилитой [swag](https://github.com/swaggo/swag):

```bash
go install github.com/swaggo/swag/cmd/swag@v1.16.4
swag init -g main.go -o docs/swagger --outputTypes json,yaml --parseInternal
```

`swagger.json` встраивается в бинарник (`go:embed`), UI — из `github.com/swaggo/files/v2`. Кнопка **Authorize** принимает `Bearer <токен>`, токен запоминается между перезагрузками страницы.

**Системные поля** (`id`, `status`, `created_by_id`, `created_at`, `published_at`) с клиента не принимаются: в структурах запросов их нет, а JSON читается с `DisallowUnknownFields`, поэтому попытка их передать — ошибка `400`.

**Статусы:** `draft → published` (метод публикации), `draft | published → deleted` (метод удаления). Вернуть запись в `draft` нельзя. Записи `deleted` клиенту не отдаются.

## HTTP-методы

Все ответы — JSON. Ошибка всегда `{"error": "текст"}`. Столбец «Доступ»: **все** — гость и пользователь, **JWT** — только с токеном, иначе `401`.

### Домен маршрутизатора `/api/provider_routers`

| # | Метод и URL | Доступ | Что делает | Ответ |
| --- | --- | --- | --- | --- |
| 1 | `GET /api/provider_routers?min_bandwidth_mbps=10000&max_cost_rub=500000` | все | Плитка: только `published`, фильтры необязательные и выполняются в БД. `is_own` = 1, если создатель — пользователь из токена (у гостя всегда 0) | `200` массив |
| 2 | `GET /api/provider_routers/feed` | все | Лента: первая опубликованная карточка. `is_liked` = 1, если пользователь из токена лайкнул (у гостя 0) | `200` / `404` |
| 2 | `GET /api/provider_routers/feed?id=2` | все | Карточка ленты по id | `200` / `404` |
| 2 | `GET /api/provider_routers/feed?id=2&next=true` | все | Следующая после id (после последней — снова первая). БД возвращает одну строку | `200` / `404` |
| 3 | `GET /api/provider_routers/draft` | JWT | Черновик пользователя из токена, id не указывается | `200` / `404` |
| 4 | `POST /api/provider_routers` | JWT | Создание черновика, создатель — из токена. `multipart/form-data`: `name` (текст), `image` (файл .jpg/.jpeg/.png/.webp/.svg до 10 МБ), `video` (файл .mp4/.webm/.mov до 50 МБ). Файлы сохраняются в MinIO под сгенерированными латинскими именами `provider_routers/router-<дата>-<hex>.png`, `videos/router-<дата>-<hex>.mp4`, имена — в `image_key`, `video_key` | `201` / `409`, если черновик уже есть |
| 5 | `PUT /api/provider_routers/{id}/publish` | JWT | Публикация своего черновика: `draft → published`, `published_at` ставит сервер. В той же транзакции заполняются `description`, `bandwidth_mbps`, `cost_rub` — их вводят на шаге «Опубликовать» (у черновика они `null`) | `200` / `403` / `404` / `409` |
| 6 | `DELETE /api/provider_routers/{id}` | JWT | Логическое удаление своей записи: `status = deleted` | `200` / `403` / `404` |
| 7 | `POST /api/provider_routers/{id}/like` | JWT | Лайк пользователя из токена: `{"like": 1}` ставит, `{"like": 0}` снимает | `200` / `404` |

Тело публикации (обязательны `description`, `bandwidth_mbps` > 0, `cost_rub` > 0). У черновика эти поля — `null`: при создании указываются только название, фото и видео, остальное заполняется на шаге публикации:

```json
{
  "description": "Конечный маршрутизатор ЖК «Озерки» на 240 квартир.",
  "bandwidth_mbps": 2500,
  "cost_rub": 84000
}
```

Элемент плитки:

```json
{
  "id": 1, "name": "Core Backbone One", "description": "…",
  "bandwidth_mbps": 100000, "cost_rub": 980000,
  "image_url": "http://localhost:9000/provider-media/provider_routers/core.svg",
  "video_url": "http://localhost:9000/provider-media/videos/core-loop.mp4",
  "likes_count": 2, "is_own": 0, "published_at": "2026-10-17T15:13:10+03:00"
}
```

Карточка ленты — те же поля, но вместо `is_own` — `is_liked` и добавлено `creator_name`. Черновик и ответы создания/публикации/удаления содержат `status`, `image_key`, `video_key`, `created_at`.

### Домен пользователя `/api/provider_router_users`

| # | Метод и URL | Доступ | Что делает | Ответ |
| --- | --- | --- | --- | --- |
| 8 | `POST /api/provider_router_users/register` | все | Регистрация: `email`, `display_name`, `password` (6–72 байта). Пароль хранится как bcrypt-хеш | `201` / `409`, если email занят |
| 9 | `POST /api/provider_router_users/login` | все | Вход: проверяет `email` + `password`, возвращает `access_token`, `token_type` = `Bearer`, `expires_in` (с), `user` | `200` / `401` |
| 10 | `POST /api/provider_router_users/logout` | JWT | Выход: токен из заголовка — в blacklist Redis до конца срока жизни | `200` / `401` |

## Таблицы БД

Каскадного удаления нет: все внешние ключи `ON UPDATE RESTRICT ON DELETE RESTRICT`.

### `provider_router_users` — пользователи

| Поле | Тип | Описание |
| --- | --- | --- |
| `id` | `BIGSERIAL` PK | |
| `email` | `VARCHAR(120)` UNIQUE NOT NULL | логин |
| `display_name` | `VARCHAR(80)` NOT NULL | имя |
| `password_hash` | `VARCHAR(72)` NOT NULL DEFAULT `''` | bcrypt-хеш пароля (добавлен в ЛР3) |

### `provider_routers` — маршрутизаторы (услуги)

| Поле | Тип | Описание |
| --- | --- | --- |
| `id` | `BIGSERIAL` PK | |
| `name` | `VARCHAR(120)` NOT NULL | наименование |
| `description` | `VARCHAR(500)` NULL | краткое описание; заполняется при публикации |
| `status` | `VARCHAR(12)` NOT NULL | `draft` / `published` / `deleted` |
| `image_key` | `VARCHAR(160)` NOT NULL | ключ изображения в MinIO |
| `video_key` | `VARCHAR(160)` NOT NULL | ключ видео в MinIO |
| `bandwidth_mbps` | `INTEGER` NULL | **поле по теме:** пропускная способность, Мбит/с; заполняется при публикации |
| `cost_rub` | `INTEGER` NULL | **поле по теме:** стоимость, ₽; заполняется при публикации |
| `created_at` | `TIMESTAMPTZ` NOT NULL | дата создания |
| `published_at` | `TIMESTAMPTZ` NULL | дата формирования (публикации) |
| `created_by_id` | `BIGINT` FK → `provider_router_users.id` | создатель |

Частичный уникальный индекс `one_draft_provider_router_per_creator (created_by_id) WHERE status = 'draft'` — не больше одного черновика у пользователя.

### `provider_router_likes` — лайки (м-м пользователь–маршрутизатор)

| Поле | Тип | Описание |
| --- | --- | --- |
| `id` | `BIGSERIAL` PK | |
| `provider_router_id` | `BIGINT` FK → `provider_routers.id` | |
| `user_id` | `BIGINT` FK → `provider_router_users.id` | |

Уникальный индекс `provider_router_like_pair (provider_router_id, user_id)` — один лайк от пользователя.

## Порядок показа

**Swagger (окно инкогнито, чтобы не было сохранённого токена).**

1. Открыть `http://localhost:8080/swagger/` → `POST /provider_router_users/login` → Try it out → `{"email": "student@provider-routers.local", "password": "password123"}` → Execute. В ответе `access_token`.
2. **Authorize** → `Bearer <access_token>` → `GET /provider_routers` — список; в запросе виден заголовок `Authorization`, у своих записей `is_own: 1`.

**Postman** (`docs/postman/provider_routers_lab4.postman_collection.json`, у коллекции авторизация `Bearer {{token}}`).

3. «01 Вход: получить JWT» — скрипт сохраняет токен в переменную `token`; дальше он подставляется во все запросы коллекции (вкладка Authorization → Inherit auth from parent).
4. «02 Список без токена (гость)» — `200`, у всех `is_own: 0`.
5. «03 Список с токеном» — `200`, `is_own` считается по пользователю из токена.
6. «04 Добавление с фото и видео» (выбрать файлы `assets/provider-media/provider_routers/core.svg` и `assets/provider-media/videos/core-loop.mp4`) → «05 Черновик» → «06 Публикация без токена» — `401` у гостя.
7. «07 Публикация с токеном» — `200`, `status: published`.
8. Остальные запросы: лента, лайк, удаление, регистрация, вход новым пользователем, «14 Выход» — токен в blacklist, «15 Публикация после выхода» — `401`.

Скрипты сами сохраняют переменные: `token` (вход), `provider_router_id` (создание/черновик), `feed_id` (лента без id), `user_email` (регистрация; email уникален за счёт `{{$timestamp}}`). При повторном показе сначала удалить или опубликовать свой черновик — иначе создание вернёт `409`.

**Redis после выхода** (cmd):

```powershell
docker compose exec redis redis-cli -a provider_routers_redis_password --scan --pattern "provider_routers:*"
docker compose exec redis redis-cli -a provider_routers_redis_password GET "provider_routers:jwt:blacklist:<токен>"
docker compose exec redis redis-cli -a provider_routers_redis_password TTL "provider_routers:jwt:blacklist:<токен>"
```

**StarUML.** Открыть `docs/provider_routers.mdj` → «Диаграмма последовательности: бизнес-процесс создателя» → выделить сообщение. В свойстве `signature` выбрана операция домена из диаграммы классов, в `arguments` — ключевые входные данные (у ответов — выходные).

**Изменения в БД** (Adminer → SQL-запрос):

```sql
SELECT id, name, status, description, image_key, video_key, bandwidth_mbps, cost_rub, created_at, published_at, created_by_id
FROM provider_routers ORDER BY id;
SELECT * FROM provider_router_likes ORDER BY id;
SELECT id, email, display_name, left(password_hash, 7) AS hash_prefix FROM provider_router_users ORDER BY id;
```

Загруженные файлы видны в MinIO Console в бакете `provider-media`.
