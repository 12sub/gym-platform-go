-- name: CountActiveMembers :one
SELECT COUNT(DISTINCT member_id)::bigint
FROM subscriptions
WHERE status = 'active'
  AND ends_on >= (now() AT TIME ZONE 'Africa/Lagos')::date;

-- name: CountExpiringSoon :one
SELECT COUNT(DISTINCT member_id)::bigint
FROM subscriptions
WHERE status = 'active'
  AND ends_on BETWEEN (now() AT TIME ZONE 'Africa/Lagos')::date
                  AND (now() AT TIME ZONE 'Africa/Lagos')::date + 7;

-- name: GetSubscription :one
SELECT * FROM subscriptions WHERE id = $1;

-- Taking this lock first serializes every change to one member's subscriptions.
-- name: LockMember :one
SELECT id, is_active FROM members WHERE id = $1 FOR UPDATE;

-- name: ListCurrentSubscriptions :many
SELECT s.id, s.status, s.starts_on, s.ends_on
FROM subscriptions s
WHERE s.member_id = sqlc.arg(member_id)::bigint
  AND s.status IN ('active', 'frozen', 'pending')
ORDER BY s.starts_on;

-- name: ExpireOverdueForMember :exec
UPDATE subscriptions
SET status = 'expired'
WHERE member_id = sqlc.arg(member_id)::bigint
  AND status = 'active'
  AND ends_on < sqlc.arg(today)::date;

-- name: CreateSubscription :one
INSERT INTO subscriptions (member_id, plan_id, starts_on, ends_on, status)
VALUES (sqlc.arg(member_id), sqlc.arg(plan_id), sqlc.arg(starts_on),
        sqlc.arg(ends_on), sqlc.arg(status))
RETURNING id;

-- name: FreezeSubscription :exec
UPDATE subscriptions
SET status = 'frozen', frozen_on = sqlc.arg(today)::date
WHERE id = sqlc.arg(id);

-- name: UnfreezeSubscription :exec
UPDATE subscriptions
SET status = 'active', ends_on = ends_on + sqlc.arg(days)::int, frozen_on = NULL
WHERE id = sqlc.arg(id);

-- name: ShiftPendingSubscriptions :exec
UPDATE subscriptions
SET starts_on = starts_on + sqlc.arg(days)::int,
    ends_on   = ends_on   + sqlc.arg(days)::int
WHERE member_id = sqlc.arg(member_id)::bigint AND status = 'pending';

-- name: CancelSubscription :exec
UPDATE subscriptions
SET status = 'cancelled', cancelled_at = now()
WHERE id = sqlc.arg(id);

-- name: ExpireOverdueSubscriptions :execrows
UPDATE subscriptions
SET status = 'expired'
WHERE status = 'active' AND ends_on < sqlc.arg(today)::date;

-- name: ActivateDueSubscriptions :execrows
UPDATE subscriptions
SET status = 'active'
WHERE status = 'pending' AND starts_on <= sqlc.arg(today)::date;

-- name: ListMemberSubscriptions :many
SELECT s.id,
       p.name AS plan_name,
       s.starts_on,
       s.ends_on,
       s.frozen_on,
       (CASE WHEN s.status = 'active' AND s.ends_on < sqlc.arg(today)::date
             THEN 'expired' ELSE s.status END)::text AS display_status
FROM subscriptions s
JOIN plans p ON p.id = s.plan_id
WHERE s.member_id = sqlc.arg(member_id)::bigint
ORDER BY s.starts_on DESC, s.id DESC;
