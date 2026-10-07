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

-- name: ExpireOverdueSubscriptions :execrows
UPDATE subscriptions
SET status = 'expired'
WHERE status = 'active'
  AND ends_on < (now() AT TIME ZONE 'Africa/Lagos')::date;
