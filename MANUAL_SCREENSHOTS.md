# Скриншоты ЛР2, которые снимаются вручную

1. Откройте `http://localhost:8081`, войдите в Adminer и покажите три таблицы: `provider_users`, `routers`, `router_likes`.
2. В Adminer выполните `SELECT id, name, status FROM routers ORDER BY id;` до и после логического удаления через кнопку плитки.
3. В браузере откройте ленту, черновик и плитку; в DevTools → Network сохраните один GET и один POST.
4. В DevTools → Response откройте HTML черновика, созданного через «Далее»: в нём должны быть `/media/routers/draft.svg` и `/media/videos/draft-loop.mp4`.
5. В редакторе покажите модели `ProviderUser`, `Router`, `RouterLike`; ORM-обработчики `publishedRouters`, `createDraftHandler`, `publishDraftHandler`; и raw SQL `UPDATE` в `routerActionHandler`.

Снимки интерфейса, которые уже были сохранены до переноса, находятся в безопасном локальном Git bundle; не добавляйте поддельные изображения вместо демонстрации работающего приложения.
