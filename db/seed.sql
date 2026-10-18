-- Обычно не нужен: сервер при первом запуске сам применяет migrations/*.up.sql и наполняет пустую БД.
-- Для ручного импорта в Adminer — после всех миграций. 104 — черновик оператора (description,
-- bandwidth_mbps, cost_rub, published_at заполняются при публикации), 105 — удалён.
INSERT INTO provider_router_users (id, email, display_name) VALUES
  (1, 'student@provider-routers.local', 'Студент РИП'),
  (2, 'operator@provider-routers.local', 'Оператор сети'),
  (3, 'viewer@provider-routers.local', 'Наблюдатель')
ON CONFLICT (id) DO NOTHING;

INSERT INTO provider_routers (id, name, description, status, image_key, video_key, bandwidth_mbps, cost_rub, created_by_id, published_at) VALUES
  (101, 'Core Backbone One', 'Центральный маршрутизатор ядра провайдера для магистрального узла и распределения трафика между сегментами сети.', 'published', 'provider_routers/core.svg', 'videos/core-loop.mp4', 100000, 980000, 2, CURRENT_TIMESTAMP),
  (102, 'North Ring Hub', 'Промежуточный маршрутизатор кольцевой сети, который агрегирует районные узлы и передаёт трафик на магистраль.', 'published', 'provider_routers/ring.svg', 'videos/ring-loop.mp4', 10000, 310000, 2, CURRENT_TIMESTAMP),
  (103, 'Harbor Residence Gateway', 'Конечный маршрутизатор жилого комплекса для распределения доступа к сети между квартирами.', 'published', 'provider_routers/residential.svg', 'videos/residential-loop.mp4', 1000, 72000, 2, CURRENT_TIMESTAMP),
  (104, 'Riverside Residence Gateway', NULL, 'draft', 'provider_routers/draft.svg', 'videos/draft-loop.mp4', NULL, NULL, 2, NULL),
  (105, 'Legacy South Edge', 'Выведенный из эксплуатации пограничный маршрутизатор.', 'deleted', 'provider_routers/ring.svg', 'videos/ring-loop.mp4', 1000, 50000, 2, CURRENT_TIMESTAMP)
ON CONFLICT (id) DO NOTHING;

INSERT INTO provider_router_likes (provider_router_id, user_id) VALUES (101, 1), (101, 3), (102, 3), (103, 1), (103, 2)
ON CONFLICT DO NOTHING;

SELECT setval(pg_get_serial_sequence('provider_router_users', 'id'), 3, true);
SELECT setval(pg_get_serial_sequence('provider_routers', 'id'), 105, true);
