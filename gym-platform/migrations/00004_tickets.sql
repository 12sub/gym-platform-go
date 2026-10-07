-- +goose Up
CREATE TABLE tickets (
  id                    BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  ticket_number         TEXT NOT NULL UNIQUE,
  title                 TEXT NOT NULL,
  description           TEXT NOT NULL,
  category              TEXT NOT NULL
                        CHECK (category IN ('equipment', 'cleanliness', 'safety', 'billing', 'other')),
  priority              TEXT NOT NULL DEFAULT 'medium'
                        CHECK (priority IN ('low', 'medium', 'high', 'urgent')),
  status                TEXT NOT NULL DEFAULT 'open'
                        CHECK (status IN ('open', 'in_progress', 'resolved', 'closed')),
  location              TEXT,
  reported_by_user_id   BIGINT REFERENCES users (id) ON DELETE SET NULL,
  reported_by_member_id BIGINT REFERENCES members (id) ON DELETE SET NULL,
  assigned_to_id        BIGINT REFERENCES users (id) ON DELETE SET NULL,
  resolved_at           TIMESTAMPTZ,
  closed_at             TIMESTAMPTZ,
  created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX tickets_status_priority_idx ON tickets (status, priority);
CREATE INDEX tickets_assigned_idx ON tickets (assigned_to_id);

CREATE TABLE ticket_comments (
  id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  ticket_id   BIGINT NOT NULL REFERENCES tickets (id) ON DELETE CASCADE,
  user_id     BIGINT REFERENCES users (id) ON DELETE SET NULL,
  body        TEXT NOT NULL,
  is_internal BOOLEAN NOT NULL DEFAULT false,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX ticket_comments_ticket_idx ON ticket_comments (ticket_id);

-- +goose Down
DROP TABLE ticket_comments;
DROP TABLE tickets;
