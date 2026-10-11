-- name: GetUserByEmail :one
SELECT u.id, u.full_name, u.email, u.password_hash, u.is_active, u.token_version,
       r.slug AS role_slug
FROM users u
JOIN roles r ON r.id = u.role_id
WHERE lower(u.email) = lower($1);

-- name: GetUserByID :one
SELECT u.id, u.full_name, u.email, u.is_active, u.token_version,
       r.slug AS role_slug
FROM users u
JOIN roles r ON r.id = u.role_id
WHERE u.id = $1;

-- name: GetRoleBySlug :one
SELECT * FROM roles WHERE slug = $1;

-- name: CreateUser :one
INSERT INTO users (full_name, email, password_hash, role_id)
VALUES ($1, $2, $3, $4)
RETURNING id;

-- name: UpsertUserByEmail :one
INSERT INTO users (full_name, email, password_hash, role_id)
VALUES ($1, $2, $3, $4)
ON CONFLICT (lower(email)) DO UPDATE
  SET full_name     = EXCLUDED.full_name,
      password_hash = EXCLUDED.password_hash,
      role_id       = EXCLUDED.role_id,
      is_active     = true,
      token_version = users.token_version + 1
RETURNING id;