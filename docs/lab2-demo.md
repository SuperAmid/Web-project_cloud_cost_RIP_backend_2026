# Порядок демонстрации ЛР2

1. Запустите `docker compose up -d`, приложение через `go run .`; покажите Adminer на `localhost:8081`.
2. В Adminer выполните `SELECT id, name, status, created_by_id FROM routers ORDER BY id;`. Покажите записи draft, published, deleted.
3. На `/routers` выполните фильтрацию `?minThroughputMbps=10000`, покажите Network с GET и сохранённое значение input.
4. Нажмите «Логически удалить» у опубликованного маршрутизатора; в Adminer выполните `SELECT id, name, status FROM routers;`. Объясните SQL `UPDATE` в `routerActionHandler`.
5. Попробуйте открыть `/routers/feed?id=<deleted-id>`: ответ 404, поскольку удалённые маршрутизаторы не отображаются.
6. На `/routers/draft` заполните только название и нажмите «Далее». В Adminer покажите созданный draft и стандартные `/media/...` URL.
7. Заполните описание, тип, пропускную способность, потребление, порты, расположение, master router и нажмите «Опубликовать». Покажите статус published в Adminer.
8. В Adminer измените числовое поле маршрутизатора и добавьте/удалите строку в `router_likes`; обновите страницу и покажите изменения характеристик/числа лайков.
9. В коде покажите `ProviderUser`, `Router`, `RouterLike`, `AutoMigrate`, partial unique index черновика и шесть handlers.
10. Откройте HTML Response всех трёх страниц и покажите default URL для изображения/видео при создании новой карточки.
