-- name: DailyRevenue :many
SELECT
  (paid_at AT TIME ZONE 'Africa/Lagos')::date AS day,
  SUM(amount)::bigint                         AS total,
  COUNT(*)::bigint                            AS payment_count
FROM payments
WHERE status = 'successful'
  AND paid_at >= sqlc.arg(start_at)::timestamptz
  AND paid_at <  sqlc.arg(end_at)::timestamptz
GROUP BY 1
ORDER BY 1;

-- name: MonthlyRevenue :many
SELECT
  date_trunc('month', paid_at AT TIME ZONE 'Africa/Lagos')::date AS month,
  SUM(amount)::bigint                                            AS total,
  COUNT(*)::bigint                                               AS payment_count
FROM payments
WHERE status = 'successful'
  AND paid_at >= sqlc.arg(start_at)::timestamptz
  AND paid_at <  sqlc.arg(end_at)::timestamptz
GROUP BY 1
ORDER BY 1;

-- name: RevenueByPlan :many
SELECT COALESCE(p.name, 'Other')::text AS plan_name,
       SUM(pay.amount)::bigint         AS total
FROM payments pay
LEFT JOIN subscriptions s ON s.id = pay.subscription_id
LEFT JOIN plans p ON p.id = s.plan_id
WHERE pay.status = 'successful'
  AND pay.paid_at >= sqlc.arg(start_at)::timestamptz
  AND pay.paid_at <  sqlc.arg(end_at)::timestamptz
GROUP BY 1
ORDER BY total DESC;

-- name: RevenueByMethod :many
SELECT method, SUM(amount)::bigint AS total
FROM payments
WHERE status = 'successful'
  AND paid_at >= sqlc.arg(start_at)::timestamptz
  AND paid_at <  sqlc.arg(end_at)::timestamptz
GROUP BY method
ORDER BY total DESC;

-- name: RevenueSummary :one
SELECT COALESCE(SUM(amount), 0)::bigint AS total,
       COUNT(*)::bigint                 AS payment_count
FROM payments
WHERE status = 'successful'
  AND paid_at >= sqlc.arg(start_at)::timestamptz
  AND paid_at <  sqlc.arg(end_at)::timestamptz;

-- name: ListPaymentsInRange :many
SELECT pay.id,
       pay.amount,
       pay.method,
       pay.paid_at,
       pay.reference,
       pay.note,
       m.id          AS member_id,
       m.full_name   AS member_name,
       m.member_code AS member_code,
       COALESCE(p.name, '')::text      AS plan_name,
       COALESCE(u.full_name, '')::text AS recorded_by_name
FROM payments pay
JOIN members m ON m.id = pay.member_id
LEFT JOIN subscriptions s ON s.id = pay.subscription_id
LEFT JOIN plans p ON p.id = s.plan_id
LEFT JOIN users u ON u.id = pay.recorded_by
WHERE pay.status = 'successful'
  AND pay.paid_at >= sqlc.arg(start_at)::timestamptz
  AND pay.paid_at <  sqlc.arg(end_at)::timestamptz
ORDER BY pay.paid_at DESC, pay.id DESC
LIMIT sqlc.arg(row_limit)::int;