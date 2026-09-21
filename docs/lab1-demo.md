# Порядок демонстрации ЛР1

1. Откройте `README.md`: покажите предметную область, вариант 15 и границы ЛР1.
2. Откройте макет в Figma и покажите три мобильных фрейма: лента, добавление, плитка; назовите референс Ubiquiti, три цвета и hover.
3. В браузере покажите `/routers/feed`: autoplay muted loop видео, характеристики, лайки, «Следующий».
4. В DevTools → Network покажите GET `/routers/feed?id=102` и затем `/routers/feed?id=102&next=true`; объясните `id` и `next`.
5. Откройте `/routers/draft`: это статус `draft`, форма не имеет сохранения и POST.
6. Откройте `/routers`: покажите две колонки, переход карточки в ленту и отсутствие удалённого узла.
7. Введите `10000` в фильтр, отправьте форму. В Network покажите GET `?minThroughputMbps=10000`, затем что значение осталось в input.
8. В DevTools → Network/Response покажите HTML ленты, черновика и плитки: URL к `localhost:9000/provider-media/...` приходят из серверной модели.
9. Откройте Minio Console и bucket `provider-media`: покажите `routers/*.svg` и предварительно загруженные `videos/*.mp4`.
10. В коде покажите `routerCollection`, методы `feedHandler`, `draftHandler`, `gridHandler`, маршрутизацию и `imageKey`/`videoKey`.

## Чек-лист скриншотов

- тема из `topics.md` и три Figma-фрейма;
- референс Ubiquiti и начало `static/styles.css`;
- три страницы приложения;
- Network с тремя GET, `id`, `next=true` и фильтром;
- HTML Response с URL Minio;
- `routerCollection`, routing и три handler;
- Minio Console с bucket и медиа.
