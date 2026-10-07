-- +goose Up

-- +goose StatementBegin
CREATE FUNCTION set_updated_at() RETURNS trigger AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TABLE roles (
  id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  name        TEXT NOT NULL,
  slug        TEXT NOT NULL UNIQUE,
  description TEXT,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO roles (name, slug, description) VALUES
  ('Owner',      'owner',      'Full access'),
  ('Manager',    'manager',    'Operations and staff'),
  ('Front Desk', 'front_desk', 'Members, check-ins, tickets'),
  ('Trainer',    'trainer',    'Member-facing, limited access'),
  ('Accountant', 'accountant', 'Revenue and payments');

CREATE TABLE users (
  id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  full_name     TEXT NOT NULL,
  email         TEXT NOT NULL,
  password_hash TEXT NOT NULL,
  role_id       BIGINT NOT NULL REFERENCES roles (id) ON DELETE RESTRICT,
  is_active     BOOLEAN NOT NULL DEFAULT true,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX users_email_lower_idx ON users (lower(email));

CREATE TABLE staff_profiles (
  id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  user_id    BIGINT NOT NULL UNIQUE REFERENCES users (id) ON DELETE CASCADE,
  phone      TEXT,
  position   TEXT,
  hired_on   DATE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Table used by the scs session manager (pgxstore)
CREATE TABLE sessions (
  token  TEXT PRIMARY KEY,
  data   BYTEA NOT NULL,
  expiry TIMESTAMPTZ NOT NULL
);

CREATE INDEX sessions_expiry_idx ON sessions (expiry);

-- +goose Down
DROP TABLE sessions;
DROP TABLE staff_profiles;
DROP TABLE users;
DROP TABLE roles;
DROP FUNCTION set_updated_at();
