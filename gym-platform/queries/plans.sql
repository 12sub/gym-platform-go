-- name: ListPlans :many
SELECT p.*,
       (SELECT COUNT(*) FROM subscriptions s
         WHERE s.plan_id = p.id AND s.status = 'active')::bigint AS active_subscribers
FROM plans p
ORDER BY p.is_active DESC, p.price_amount ASC;

-- name: GetPlan :one
SELECT * FROM plans WHERE id = $1;

-- name: CreatePlan :one
INSERT INTO plans (name, description, price_amount, duration_days)
VALUES ($1, $2, $3, $4)
RETURNING id;

-- name: UpdatePlan :execrows
UPDATE plans
SET name = $2, description = $3, price_amount = $4, duration_days = $5
WHERE id = $1;

-- name: SetPlanActive :exec
UPDATE plans SET is_active = $2 WHERE id = $1;
