-- +goose Up
CREATE TABLE members (
  id                      BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  member_code             TEXT NOT NULL UNIQUE,
  full_name               TEXT NOT NULL,
  email                   TEXT,
  phone                   TEXT NOT NULL,
  date_of_birth           DATE,
  emergency_contact_name  TEXT,
  emergency_contact_phone TEXT,
  joined_on               DATE NOT NULL,
  is_active               BOOLEAN NOT NULL DEFAULT true,
  created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX members_email_lower_idx ON members (lower(email)) WHERE email IS NOT NULL;

CREATE TABLE plans (
  id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  name          TEXT NOT NULL,
  description   TEXT,
  price_amount  BIGINT NOT NULL CHECK (price_amount >= 0),   -- kobo
  duration_days INTEGER NOT NULL CHECK (duration_days > 0),
  is_active     BOOLEAN NOT NULL DEFAULT true,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE subscriptions (
  id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  member_id    BIGINT NOT NULL REFERENCES members (id) ON DELETE CASCADE,
  plan_id      BIGINT NOT NULL REFERENCES plans (id) ON DELETE RESTRICT,
  starts_on    DATE NOT NULL,
  ends_on      DATE NOT NULL,
  status       TEXT NOT NULL DEFAULT 'pending'
               CHECK (status IN ('pending', 'active', 'frozen', 'cancelled', 'expired')),
  cancelled_at TIMESTAMPTZ,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  CHECK (ends_on >= starts_on)
);

CREATE INDEX subscriptions_status_ends_on_idx ON subscriptions (status, ends_on);
CREATE INDEX subscriptions_member_idx ON subscriptions (member_id);

-- +goose Down
DROP TABLE subscriptions;
DROP TABLE plans;
DROP TABLE members;
