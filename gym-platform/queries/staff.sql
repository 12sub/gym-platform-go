-- name: ListStaff :many
SELECT u.id, u.full_name, u.email, u.is_active,
       (u.password_hash = '!')::bool        AS pending_setup,
       r.slug                               AS role_slug,
       r.name                               AS role_name,
       COALESCE(sp.phone, '')::text         AS phone,
       COALESCE(sp.position, '')::text      AS position
FROM users u
JOIN roles r ON r.id = u.role_id
LEFT JOIN staff_profiles sp ON sp.user_id = u.id
ORDER BY u.is_active DESC, lower(u.full_name);

-- name: GetStaff :one
SELECT u.id, u.full_name, u.email, u.is_active,
       (u.password_hash = '!')::bool        AS pending_setup,
       r.slug                               AS role_slug,
       r.name                               AS role_name,
       COALESCE(sp.phone, '')::text         AS phone,
       COALESCE(sp.position, '')::text      AS position
FROM users u
JOIN roles r ON r.id = u.role_id
LEFT JOIN staff_profiles sp ON sp.user_id = u.id
WHERE u.id = $1;

-- Held for the length of a transaction so two staff changes can't race past
-- the "at least one owner" check.
-- name: LockStaffChanges :exec
SELECT pg_advisory_xact_lock(7001);

-- name: CountActiveOwnersExcept :one
SELECT COUNT(*)::bigint
FROM users u
JOIN roles r ON r.id = u.role_id
WHERE r.slug = 'owner' AND u.is_active AND u.id <> sqlc.arg(user_id)::bigint;

-- name: UpdateStaffUser :execrows
UPDATE users
SET full_name = sqlc.arg(full_name), role_id = sqlc.arg(role_id)
WHERE id = sqlc.arg(id);

-- name: UpsertStaffProfile :exec
INSERT INTO staff_profiles (user_id, phone, position)
VALUES (sqlc.arg(user_id), sqlc.narg(phone), sqlc.narg(position))
ON CONFLICT (user_id) DO UPDATE
  SET phone = EXCLUDED.phone, position = EXCLUDED.position;

-- name: SetUserActive :exec
UPDATE users SET is_active = sqlc.arg(is_active) WHERE id = sqlc.arg(id);

-- name: GetPasswordHash :one
SELECT password_hash FROM users WHERE id = $1;

-- Changing a password bumps token_version, which signs the user out everywhere.
-- name: SetPassword :one
UPDATE users
SET password_hash = sqlc.arg(password_hash), token_version = token_version + 1
WHERE id = sqlc.arg(id)
RETURNING token_version;

-- name: CreatePasswordToken :exec
INSERT INTO password_tokens (user_id, token_hash, purpose, expires_at, created_by)
VALUES (sqlc.arg(user_id), sqlc.arg(token_hash), sqlc.arg(purpose),
        sqlc.arg(expires_at), sqlc.narg(created_by));

-- name: GetUsablePasswordToken :one
SELECT pt.id, pt.user_id, pt.purpose, u.full_name, u.email, u.is_active
FROM password_tokens pt
JOIN users u ON u.id = pt.user_id
WHERE pt.token_hash = $1 AND pt.used_at IS NULL AND pt.expires_at > now();

-- name: MarkPasswordTokenUsed :execrows
UPDATE password_tokens SET used_at = now() WHERE id = $1 AND used_at IS NULL;

-- name: InvalidateUserTokens :exec
UPDATE password_tokens SET used_at = now() WHERE user_id = $1 AND used_at IS NULL;
