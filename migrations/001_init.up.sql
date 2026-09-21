CREATE TABLE provider_users (
  id BIGSERIAL PRIMARY KEY,
  email VARCHAR(120) NOT NULL UNIQUE,
  display_name VARCHAR(80) NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE routers (
  id BIGSERIAL PRIMARY KEY,
  name VARCHAR(120) NOT NULL,
  description VARCHAR(500) NOT NULL DEFAULT '',
  status VARCHAR(12) NOT NULL CHECK (status IN ('draft', 'published', 'deleted')),
  image_url VARCHAR(500) NOT NULL DEFAULT '',
  video_url VARCHAR(500) NOT NULL DEFAULT '',
  router_type VARCHAR(24) NOT NULL DEFAULT 'residential',
  throughput_mbps INTEGER NOT NULL DEFAULT 0 CHECK (throughput_mbps >= 0),
  power_consumption_w INTEGER NOT NULL DEFAULT 0 CHECK (power_consumption_w >= 0),
  port_count INTEGER NOT NULL DEFAULT 0 CHECK (port_count >= 0),
  location VARCHAR(160) NOT NULL DEFAULT '',
  master_router_name VARCHAR(120) NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  published_at TIMESTAMPTZ,
  created_by_id BIGINT NOT NULL REFERENCES provider_users(id) ON UPDATE RESTRICT ON DELETE RESTRICT
);

CREATE INDEX routers_status_idx ON routers(status);
CREATE INDEX routers_creator_idx ON routers(created_by_id);
CREATE UNIQUE INDEX one_draft_router_per_creator ON routers(created_by_id) WHERE status = 'draft';

CREATE TABLE router_likes (
  router_id BIGINT NOT NULL REFERENCES routers(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
  user_id BIGINT NOT NULL REFERENCES provider_users(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (router_id, user_id)
);
