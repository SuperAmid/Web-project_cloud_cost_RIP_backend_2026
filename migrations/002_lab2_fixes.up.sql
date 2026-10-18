-- ЛР2, исправления по замечаниям:
--   * лишнее поле router_type удалено (по теме только два поля);
--   * четыре поля, которые заполняются на шаге «Опубликовать», необязательные (NULL);
--   * created_at оставлен только у маршрутизатора;
--   * у лайков собственный первичный ключ id и уникальная пара (маршрутизатор, пользователь).
ALTER TABLE provider_routers DROP COLUMN IF EXISTS router_type;

ALTER TABLE provider_routers
  ALTER COLUMN description DROP NOT NULL,
  ALTER COLUMN description DROP DEFAULT,
  ALTER COLUMN bandwidth_mbps DROP NOT NULL,
  ALTER COLUMN bandwidth_mbps DROP DEFAULT,
  ALTER COLUMN cost_rub DROP NOT NULL,
  ALTER COLUMN cost_rub DROP DEFAULT;

UPDATE provider_routers SET description = NULL WHERE status = 'draft' AND description = '';

ALTER TABLE provider_router_users DROP COLUMN IF EXISTS created_at;
ALTER TABLE provider_router_likes DROP COLUMN IF EXISTS created_at;

ALTER TABLE provider_router_likes DROP CONSTRAINT IF EXISTS provider_router_likes_pkey;
ALTER TABLE provider_router_likes ADD COLUMN IF NOT EXISTS id BIGSERIAL;
ALTER TABLE provider_router_likes ADD PRIMARY KEY (id);
CREATE UNIQUE INDEX IF NOT EXISTS provider_router_like_pair
  ON provider_router_likes (provider_router_id, user_id);
