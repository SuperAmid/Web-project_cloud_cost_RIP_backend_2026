# Provider Router Network — РИП ЛР1

Лабораторная работа №1 по предметной области «Электронные устройства и электричество», вариант 15. Интерфейс показывает маршрутизаторы провайдера: центральный, промежуточный и жилые конечные узлы. Расчёт нагрузки и заявки намеренно не реализованы: это задачи следующих лабораторных.

## Реализовано

- Go `net/http` и серверные HTML-шаблоны, без БД, ORM, JavaScript и POST.
- Одна in-memory коллекция `Router`: черновик, опубликованные и удалённый маршрутизатор; лайки вычисляются из `LikedUserIDs` на сервере.
- Ровно три основных GET-маршрута: лента, черновик, плитка с серверным числовым фильтром.
- Mobile-first CSS в отдельном файле, общая навигация из трёх вкладок.
- Docker Compose с Minio и инициализацией публичного bucket `provider-media`.

## Запуск

Требования: Go 1.24+ и Docker Desktop.

```powershell
docker compose up -d
go run .
```

Откройте `http://localhost:8080/routers/feed`. Minio Console: `http://localhost:9001` (`minioadmin` / `minioadmin`, только для локальной демонстрации).

Перед демонстрацией загрузите четыре коротких MP4 с ключами из `assets/provider-media/README.md` в `provider-media/videos/`; SVG-превью и структура bucket загружаются автоматически.

## Проверка GET

| Что | URL |
| --- | --- |
| Лента, первый опубликованный узел | `http://localhost:8080/routers/feed` |
| Лента по ID | `http://localhost:8080/routers/feed?id=102` |
| Следующий опубликованный узел | `http://localhost:8080/routers/feed?id=102&next=true` |
| Черновик | `http://localhost:8080/routers/draft` |
| Плитка | `http://localhost:8080/routers` |
| Фильтр пропускной способности | `http://localhost:8080/routers?minThroughputMbps=10000` |

В demo-данных: 101 — центральный, 102 — промежуточный, 103 — жилой опубликованные маршрутизаторы; 104 — черновик; 105 — удалённый и в UI не выводится.

## Media URL

По умолчанию URL строятся из `http://localhost:9000/provider-media`. Для другого хоста Minio задайте `MINIO_PUBLIC_URL`, например:

```powershell
$env:MINIO_PUBLIC_URL = 'http://localhost:9000/provider-media'
go run .
```
