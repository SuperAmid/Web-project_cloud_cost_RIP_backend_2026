# ЛР‑2: проверяемые сценарии

- GET `/provider_routers/feed`, `id`, `next=true`;
- GET `/provider_routers?minBandwidthMbps=10000`: серверная фильтрация по пропускной способности;
- POST `/provider_routers/draft/create`: GORM создаёт черновик;
- POST `/provider_routers/draft/publish`: GORM публикует черновик;
- POST `/provider_routers/{id}/delete`: параметризованный SQL `UPDATE` меняет статус на `deleted`.

Во всех экранах применяются два тематических числа: пропускная способность и стоимость. Фото и видео используют отдельные ключи медиа в MinIO.
