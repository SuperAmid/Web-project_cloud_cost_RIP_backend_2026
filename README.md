# provider_routers — ЛР‑2

Лабораторная работа №2: PostgreSQL, GORM и серверные шаблоны по теме «Маршрутизаторы сети провайдера». Единое имя проекта, таблиц, модели и URL — `provider_routers`.

## Предметная область и интерфейс

У маршрутизатора два параметра по теме: пропускная способность (Мбит/с) и стоимость (₽). Стиль взят с сайта провайдера «Ростелеком» (rt.ru): фиолетовая навигация `#7700FF`, текст `#101828`, белые карточки с серой обводкой; все кнопки красные `#E30613`, прямоугольные со скруглением 5 px (красный — по требованию преподавателя вместо фирменного `#FF4F12`).

- **Лента** — одна опубликованная запись; из БД приходит ровно одна строка (`First` → `LIMIT 1`). Описание в две строки с нативным «Ещё», справа — параметры, лайк и «Следующий».
- **Добавление, шаг 1 («Далее»)** — только три поля: фото, видео (выбор из проводника) и название. Файлы на сервер не передаются; в таблицу записываются неверные ключи медиа (`provider_routers/not-uploaded.jpg`, `videos/not-uploaded.mp4`).
- **Добавление, шаг 2 («Опубликовать»)** — если черновик уже есть, страница открывается с ним: фото и видео, название текстом, краткое описание и два поля по теме.
- **Фото и видео по умолчанию** лежат на SSR-сервере (`static/media/`) и показываются, если ключ пустой или файл по ссылке в MinIO недоступен — на обоих шагах добавления и на всех страницах.
- **Маршрутизаторы** — плитка опубликованных записей, фильтр слайдером на сервере, кнопка «Удалить».
- Лайки только отображаются из БД.

## Где что лежит

| Файл | Что внутри |
| --- | --- |
| `main.go` | маршруты и 6 обработчиков: три GET и три POST |
| `models.go` | модели GORM — по одной структуре на таблицу |
| `media.go` | ключ → URL MinIO или файл по умолчанию с SSR-сервера |
| `migrate.go` | выполнение версионированных миграций `migrations/*.up.sql` при запуске |
| `migrations/001_init.up.sql` | исходная схема трёх таблиц |
| `migrations/002_lab2_fixes.up.sql` | исправления: 4 необязательных поля, `created_at` только у услуги, `id` у лайков, без `router_type` |
| `db/seed.sql` | начальные данные для импорта в Adminer |
| `templates/feed.html` | страница «Лента» |
| `templates/draft.html` | страница «Добавление» (шаги «Далее» и «Опубликовать») |
| `templates/grid.html` | страница «Маршрутизаторы» (плитка, фильтр, «Удалить») |
| `templates/partials.html` | нижняя навигация и строка состояния телефона (время, камера, заряд) |
| `templates/notfound.html` | страница 404 (удалённая карточка) |
| `static/styles.css` | все стили |
| `static/media/` | фото и видео по умолчанию на SSR-сервере |
| `assets/provider-media/` | файлы, которые `minio-init` загружает в бакет MinIO |
| `docs/lab2-er-diagram.mdj` | ER-диаграмма в StarUML (+ экспорт `docs/lab2-er-diagram.png`) |
| `docs/lab2-fixes.md` | исправления по замечаниям |

## Запуск

```powershell
docker compose up -d
go run .
```

Откройте `http://localhost:8080/provider_routers/feed`.

При запуске приложение выполняет миграции из `migrations/` (выполненные версии — в таблице `schema_migrations`) и наполняет пустую БД начальными данными.

- PostgreSQL: `localhost:5433`, БД `provider_routers`, пользователь `provider_routers_user`, пароль `provider_routers_password`.
- Adminer: `http://localhost:8081` (сервер `postgres`).
- MinIO Console: `http://localhost:9001` (`minioadmin` / `minioadmin`), бакет `provider-media`. Образы — `bitnamilegacy/minio` и `bitnamilegacy/minio-client` по digest (официальные `quay.io/minio/*` стали приватными), у MinIO есть `healthcheck`.

## HTTP-методы

| Метод | URL | Как работает |
| --- | --- | --- |
| GET | `/provider_routers/feed`, `?id=102`, `?id=102&next=true` | ORM, `Preload("Likes")…First()` — одна строка |
| GET | `/provider_routers/draft` | ORM, черновик текущего пользователя |
| GET | `/provider_routers?minBandwidthMbps=10000` | ORM, фильтр `WHERE bandwidth_mbps >= ?` |
| POST | `/provider_routers/draft/create` | ORM `Create`: название + неверные ключи медиа, статус `draft` |
| POST | `/provider_routers/draft/publish` | ORM `Updates`: описание, два поля по теме, `published_at`, статус `published` |
| POST | `/provider_routers/{id}/delete` | **без ORM**: `UPDATE provider_routers SET status = 'deleted' … RETURNING id` через `database/sql`, результат читается курсором `rows.Next()` |

## Таблицы

| Таблица | Поля |
| --- | --- |
| `provider_router_users` | `id` PK, `email` UNIQUE, `display_name` |
| `provider_routers` | `id` PK, `name`, `description` NULL, `status`, `image_key`, `video_key`, `bandwidth_mbps` NULL, `cost_rub` NULL, `created_at`, `published_at` NULL, `created_by_id` FK |
| `provider_router_likes` | `id` PK, `provider_router_id` FK, `user_id` FK, уникальная пара (`provider_router_id`, `user_id`) |

Каскадного удаления нет: все внешние ключи `ON UPDATE RESTRICT ON DELETE RESTRICT`. Частичный уникальный индекс `one_draft_provider_router_per_creator` — не больше одного черновика у пользователя.

## Порядок показа (скриншоты `docs/screenshots/`)

Сняты на чистой БД после `docker compose up -d` и `go run .`. SQL-результаты — `psql` (в Adminer те же запросы: «SQL-запрос»).

| № | Файл | Что показывает |
| --- | --- | --- |
| 1 | `01-sql-start.png` | начальные данные: три опубликованных, черновик (NULL в 4 необязательных полях, неверные ключи медиа), удалённый |
| 2 | `02-sql-likes.png` | лайки: собственный `id`, пара (маршрутизатор, пользователь) уникальна |
| 3 | `03-sql-users.png` | пользователи без `created_at` |
| 4 | `04-feed.png` | лента: первая опубликованная карточка, параметры и иконки справа |
| 5 | `05-feed-next.png` | `?id=1&next=true` — следующая карточка (БД возвращает одну строку `LIMIT 1`) |
| 6 | `06-feed-wrap.png` | после последней — снова первая |
| 7 | `07-grid.png` | плитка, кнопки «Удалить» |
| 8 | `08-grid-filter.png` | фильтр слайдером `minBandwidthMbps=10000` (фильтрация в БД) |
| 9 | `09-draft-publish-form.png` | черновик есть → «Опубликовать»: фото и видео по умолчанию с SSR, название текстом, описание и два поля по теме |
| 10 | `10-after-publish.png` | опубликованная карточка в ленте |
| 11 | `11-sql-published.png` | в БД заполнены `description`, `bandwidth_mbps`, `cost_rub`, `published_at`, статус `published` |
| 12 | `12-draft-step1.png` | черновика нет → «Далее»: фото, видео (по умолчанию) и только название |
| 13 | `13-after-next-defaults.png` | после «Далее»: черновик создан, медиа по умолчанию |
| 14 | `14-sql-created.png` | новый черновик: только название, неверные ключи `not-uploaded.*`, остальное NULL |
| 15 | `15-after-delete.png` | «Удалить» на плитке — карточка пропала |
| 16 | `16-sql-deleted.png` | статус `deleted` (UPDATE … RETURNING через курсор `rows.Next()`) |
| 17 | `17-feed-deleted-404.png` | удалённую карточку открыть нельзя — `404` |
