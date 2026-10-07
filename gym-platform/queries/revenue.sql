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
SELECT p.name AS plan_name, SUM(pay.amount)::bigint AS total
FROM payments pay
JOIN subscriptions s ON s.id = pay.subscription_id
JOIN plans p ON p.id = s.plan_id
WHERE pay.status = 'successful'
  AND pay.paid_at >= sqlc.arg(start_at)::timestamptz
  AND pay.paid_at <  sqlc.arg(end_at)::timestamptz
GROUP BY p.name
ORDER BY total DESC;