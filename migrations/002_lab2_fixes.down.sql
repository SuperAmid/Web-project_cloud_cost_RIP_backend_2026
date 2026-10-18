DROP INDEX IF EXISTS provider_router_like_pair;
ALTER TABLE provider_router_likes DROP CONSTRAINT IF EXISTS provider_router_likes_pkey;
ALTER TABLE provider_router_likes DROP COLUMN IF EXISTS id;
ALTER TABLE provider_router_likes ADD PRIMARY KEY (provider_router_id, user_id);
ALTER TABLE provider_router_likes ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE provider_router_users ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP;

UPDATE provider_routers SET description = '' WHERE description IS NULL;
UPDATE provider_routers SET bandwidth_mbps = 0 WHERE bandwidth_mbps IS NULL;
UPDATE provider_routers SET cost_rub = 0 WHERE cost_rub IS NULL;
ALTER TABLE provider_routers
  ALTER COLUMN description SET DEFAULT '',
  ALTER COLUMN description SET NOT NULL,
  ALTER COLUMN bandwidth_mbps SET DEFAULT 0,
  ALTER COLUMN bandwidth_mbps SET NOT NULL,
  ALTER COLUMN cost_rub SET DEFAULT 0,
  ALTER COLUMN cost_rub SET NOT NULL;
ALTER TABLE provider_routers ADD COLUMN IF NOT EXISTS router_type VARCHAR(24) NOT NULL DEFAULT 'residential';
