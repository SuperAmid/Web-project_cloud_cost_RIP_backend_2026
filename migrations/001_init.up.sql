CREATE TABLE provider_router_users (
  id BIGSERIAL PRIMARY KEY,
  email VARCHAR(120) NOT NULL UNIQUE,
  display_name VARCHAR(80) NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE provider_routers (
  id BIGSERIAL PRIMARY KEY,
  name VARCHAR(120) NOT NULL,
  description VARCHAR(500) NOT NULL DEFAULT '',
  status VARCHAR(12) NOT NULL CHECK (status IN ('draft', 'published', 'deleted')),
  image_key VARCHAR(160) NOT NULL DEFAULT '',
  video_key VARCHAR(160) NOT NULL DEFAULT '',
  router_type VARCHAR(24) NOT NULL DEFAULT 'residential',
  bandwidth_mbps INTEGER NOT NULL DEFAULT 0 CHECK (bandwidth_mbps >= 0),
  cost_rub INTEGER NOT NULL DEFAULT 0 CHECK (cost_rub >= 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  published_at TIMESTAMPTZ,
  created_by_id BIGINT NOT NULL REFERENCES provider_router_users(id) ON UPDATE RESTRICT ON DELETE RESTRICT
);

CREATE INDEX provider_routers_status_idx ON provider_routers(status);
CREATE INDEX provider_routers_creator_idx ON provider_routers(created_by_id);
CREATE UNIQUE INDEX one_draft_provider_router_per_creator ON provider_routers(created_by_id) WHERE status = 'draft';

CREATE TABLE provider_router_likes (
  provider_router_id BIGINT NOT NULL REFERENCES provider_routers(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
  user_id BIGINT NOT NULL REFERENCES provider_router_users(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (provider_router_id, user_id)
);
