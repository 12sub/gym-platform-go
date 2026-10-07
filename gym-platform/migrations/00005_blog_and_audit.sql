-- +goose Up
CREATE TABLE posts (
  id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  title        TEXT NOT NULL,
  slug         TEXT NOT NULL UNIQUE,
  excerpt      TEXT,
  body         TEXT NOT NULL,
  cover_image  TEXT,
  status       TEXT NOT NULL DEFAULT 'draft'
               CHECK (status IN ('draft', 'scheduled', 'published')),
  scheduled_at TIMESTAMPTZ,
  published_at TIMESTAMPTZ,
  author_id    BIGINT REFERENCES users (id) ON DELETE SET NULL,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX posts_status_scheduled_idx ON posts (status, scheduled_at);

CREATE TABLE social_publications (
  id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  post_id      BIGINT NOT NULL REFERENCES posts (id) ON DELETE CASCADE,
  platform     TEXT NOT NULL CHECK (platform IN ('facebook', 'instagram', 'x', 'linkedin')),
  status       TEXT NOT NULL DEFAULT 'pending'
               CHECK (status IN ('pending', 'published', 'failed')),
  external_id  TEXT,
  error        TEXT,
  attempts     INTEGER NOT NULL DEFAULT 0,
  published_at TIMESTAMPTZ,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (post_id, platform)
);

CREATE TABLE audit_logs (
  id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  user_id     BIGINT REFERENCES users (id) ON DELETE SET NULL,
  action      TEXT NOT NULL,                  -- 'role.changed', 'payment.refunded'
  entity_type TEXT,
  entity_id   BIGINT,
  metadata    JSONB,
  ip_address  TEXT,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX audit_logs_entity_idx ON audit_logs (entity_type, entity_id);

-- +goose Down
DROP TABLE audit_logs;
DROP TABLE social_publications;
DROP TABLE posts;
