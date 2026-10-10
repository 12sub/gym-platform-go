-- name: CreatePayment :exec
INSERT INTO payments (member_id, subscription_id, recorded_by, amount, method,
                      status, reference, note, paid_at)
VALUES (sqlc.arg(member_id), sqlc.narg(subscription_id), sqlc.narg(recorded_by),
        sqlc.arg(amount), sqlc.arg(method), 'successful', sqlc.arg(reference),
        sqlc.narg(note), now());

-- name: ListMemberPayments :many
SELECT pay.id,
       pay.amount,
       pay.method,
       pay.status,
       pay.reference,
       pay.paid_at,
       pay.note,
       COALESCE(p.name, '')::text      AS plan_name,
       COALESCE(u.full_name, '')::text AS recorded_by_name
FROM payments pay
LEFT JOIN subscriptions s ON s.id = pay.subscription_id
LEFT JOIN plans p ON p.id = s.plan_id
LEFT JOIN users u ON u.id = pay.recorded_by
WHERE pay.member_id = sqlc.arg(member_id)::bigint
ORDER BY pay.created_at DESC, pay.id DESC
LIMIT 50;

-- name: RevenueBetween :one
SELECT COALESCE(SUM(amount), 0)::bigint
FROM payments
WHERE status = 'successful'
  AND paid_at >= sqlc.arg(start_at)::timestamptz
  AND paid_at <  sqlc.arg(end_at)::timestamptz;
