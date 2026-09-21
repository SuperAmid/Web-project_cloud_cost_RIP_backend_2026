-- Импортируйте файл в Adminer после migrations/001_init.up.sql.
INSERT INTO provider_users (id, email, display_name) VALUES
  (1, 'student@provider-network.local', 'Студент РИП'),
  (2, 'operator@provider-network.local', 'Оператор сети'),
  (3, 'viewer@provider-network.local', 'Наблюдатель')
ON CONFLICT (id) DO NOTHING;

INSERT INTO routers (id, name, description, status, image_url, video_url, router_type, throughput_mbps, power_consumption_w, port_count, location, master_router_name, created_by_id, published_at) VALUES
  (101, 'Core Backbone One', 'Центральный маршрутизатор ядра провайдера для магистрального узла.', 'published', 'http://localhost:9000/provider-media/routers/core.svg', 'http://localhost:9000/provider-media/videos/core-loop.mp4', 'central', 100000, 1450, 8, 'Центральный ЦОД, стойка A-12', '—', 2, CURRENT_TIMESTAMP),
  (102, 'North Ring Hub', 'Промежуточный маршрутизатор кольцевой сети с агрегацией районных узлов.', 'published', 'http://localhost:9000/provider-media/routers/ring.svg', 'http://localhost:9000/provider-media/videos/ring-loop.mp4', 'intermediate', 10000, 310, 24, 'Северный узел', 'Core Backbone One', 2, CURRENT_TIMESTAMP),
  (103, 'Harbor Residence Gateway', 'Конечный маршрутизатор жилого комплекса: распределение трафика квартир.', 'published', 'http://localhost:9000/provider-media/routers/residential.svg', 'http://localhost:9000/provider-media/videos/residential-loop.mp4', 'residential', 1000, 72, 16, 'Жилой комплекс «Панорама», корпус 3', 'North Ring Hub', 2, CURRENT_TIMESTAMP),
  (104, 'Riverside Residence Gateway', 'Черновик карточки маршрутизатора для следующего жилого дома.', 'draft', '', '', 'residential', 1000, 48, 12, 'Жилой комплекс «Речной», корпус 1', 'North Ring Hub', 2, NULL),
  (105, 'Legacy South Edge', 'Выведенный из эксплуатации пограничный маршрутизатор.', 'deleted', '', '', 'intermediate', 1000, 210, 8, 'Южный узел', 'Core Backbone One', 2, NULL)
ON CONFLICT (id) DO NOTHING;

INSERT INTO router_likes (router_id, user_id) VALUES (101, 1), (101, 3), (102, 3), (103, 1), (103, 2)
ON CONFLICT DO NOTHING;

SELECT setval(pg_get_serial_sequence('provider_users', 'id'), 3, true);
SELECT setval(pg_get_serial_sequence('routers', 'id'), 105, true);
