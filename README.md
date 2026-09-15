# Provider Router Network — РИП ЛР2

Лабораторная работа №2 по предметной области «Электронные устройства и электричество», вариант 15. Интерфейс показывает маршрутизаторы провайдера: центральный, промежуточный и жилые конечные узлы. Расчёт нагрузки и заявки намеренно не реализованы: это задачи следующих лабораторных.

## Реализовано

- Go `net/http`, GORM, PostgreSQL и серверные HTML-шаблоны; JavaScript отсутствует.
- Три предметные таблицы: `routers`, `provider_users`, `router_likes`; миграция выполняется через GORM при запуске.
- Ровно три GET-маршрута: лента, черновик, плитка с серверным числовым фильтром; три POST: создание черновика, публикация, логическое удаление.
- Создание и публикация используют ORM; удаление выполняется явным SQL `UPDATE` статуса.
- Mobile-first CSS в отдельном файле, общая навигация из трёх вкладок.
- Docker Compose с Minio и инициализацией публичного bucket `provider-media`.

## Запуск

Требования: Go 1.24+ и Docker Desktop.

```powershell
docker compose up -d
go run .
```

Откройте `http://localhost:8080/routers/feed`. Adminer: `http://localhost:8081` (System: PostgreSQL, Server: `postgres`, User: `router_user`, Password: `router_password`, Database: `provider_network`). Minio Console: `http://localhost:9001` (`minioadmin` / `minioadmin`, только для локальной демонстрации).

Четыре коротких MP4 и SVG-превью хранятся в `assets/provider-media/`; команда `docker compose up -d` автоматически загружает их в bucket `provider-media`.

## Проверка GET

| Что | URL |
| --- | --- |
| Лента, первый опубликованный узел | `http://localhost:8080/routers/feed` |
| Лента по ID | `http://localhost:8080/routers/feed?id=102` |
| Следующий опубликованный узел | `http://localhost:8080/routers/feed?id=102&next=true` |
| Черновик | `http://localhost:8080/routers/draft` |
| Плитка | `http://localhost:8080/routers` |
| Фильтр пропускной способности | `http://localhost:8080/routers?minThroughputMbps=10000` |

POST-сценарии: откройте `/routers/draft`, укажите название и нажмите «Далее»; затем заполните черновик и нажмите «Опубликовать». На странице плитки используйте «Логически удалить». Новые URL изображения и видео в ЛР2 намеренно не отправляются: используется SSR-медиа по умолчанию.

В demo-данных: 101 — центральный, 102 — промежуточный, 103 — жилой опубликованные маршрутизаторы; 104 — черновик; 105 — удалённый и в UI не выводится.

## Media URL

По умолчанию URL строятся из `http://localhost:9000/provider-media`. Для другого хоста Minio задайте `MINIO_PUBLIC_URL`, например:

```powershell
$env:MINIO_PUBLIC_URL = 'http://localhost:9000/provider-media'
go run .
```
