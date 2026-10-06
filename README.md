# provider_routers — ЛР‑2

Лабораторная работа №2: PostgreSQL, GORM и серверные шаблоны для темы «Маршрутизаторы провайдера». Единое имя проекта, таблиц, модели и URL — `provider_routers`.

## Предметная область и интерфейс

У маршрутизатора только два тематических параметра: пропускная способность (Мбит/с) и стоимость (₽). Все кнопки красные и с небольшим скруглением; карточки белые с серой границей; нижняя навигация фиолетовая.

- лента показывает опубликованную запись, короткое описание в две строки и нативное «Ещё»;
- плитка фильтруется слайдером минимальной пропускной способности;
- в добавлении доступны фото и видео из проводника, но ЛР‑2 не загружает их: используются отдельные ключи медиа по умолчанию из MinIO;
- новый черновик и публикация выполняются через GORM; логическое удаление — явным SQL `UPDATE`.

## Запуск

```powershell
docker compose up -d
go run .
```

Откройте `http://localhost:8080/provider_routers/feed`.

PostgreSQL: `localhost:5433`, БД `provider_routers`, пользователь `provider_routers_user`, пароль `provider_routers_password`. Adminer: `http://localhost:8081`. MinIO Console: `http://localhost:9001` (`minioadmin` / `minioadmin`).

## Миграции и модель

Версионированные SQL-файлы лежат в `migrations/`, начальные данные — в `db/seed.sql`. В БД три таблицы:

- `provider_router_users`;
- `provider_routers`;
- `provider_router_likes`.

В `provider_routers` отдельно хранятся `image_key` и `video_key`; связи защищены внешними ключами `RESTRICT`. Частичный уникальный индекс допускает только один `draft` на создателя.

## Маршруты

| Экран или действие | URL |
| --- | --- |
| Лента | `/provider_routers/feed` |
| Лента по ID | `/provider_routers/feed?id=102` |
| Следующий | `/provider_routers/feed?id=102&next=true` |
| Добавление/черновик | `/provider_routers/draft` |
| Создать черновик | `POST /provider_routers/draft/create` |
| Опубликовать | `POST /provider_routers/draft/publish` |
| Плитка и фильтр | `/provider_routers?minBandwidthMbps=10000` |
| Логически удалить | `POST /provider_routers/{id}/delete` |

## Проверка

```powershell
go test -count=1 ./...
```

ER-описание: `docs/lab2-er-diagram.md`; SVG-схема: `docs/lab2-er-diagram.svg`. Для сдачи в StarUML нужно сохранить настоящий `.mdj` — SVG не является его заменой.
