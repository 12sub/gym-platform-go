-- +goose Up
CREATE TABLE payments (
  id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  member_id       BIGINT NOT NULL REFERENCES members (id) ON DELETE RESTRICT,
  subscription_id BIGINT REFERENCES subscriptions (id) ON DELETE SET NULL,
  recorded_by     BIGINT REFERENCES users (id) ON DELETE SET NULL,
  amount          BIGINT NOT NULL CHECK (amount > 0),         -- kobo
  currency        CHAR(3) NOT NULL DEFAULT 'NGN',
  method          TEXT NOT NULL CHECK (method IN ('card', 'transfer', 'cash', 'pos')),
  status          TEXT NOT NULL DEFAULT 'pending'
                  CHECK (status IN ('pending', 'successful', 'failed', 'refunded')),
  gateway         TEXT,
  reference       TEXT NOT NULL UNIQUE,                       -- idempotency key
  paid_at         TIMESTAMPTZ,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX payments_status_paid_at_idx ON payments (status, paid_at);
CREATE INDEX payments_member_idx ON payments (member_id);

-- +goose Down
DROP TABLE payments;
