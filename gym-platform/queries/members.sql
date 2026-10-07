-- name: ListMembers :many
SELECT m.id,
       m.member_code,
       m.full_name,
       m.email,
       m.phone,
       m.is_active,
       COALESCE(cs.plan_name, '')::text AS plan_name,
       (CASE
          WHEN cs.status IS NULL THEN 'none'
          WHEN cs.status = 'active'
               AND cs.ends_on < (now() AT TIME ZONE 'Africa/Lagos')::date THEN 'expired'
          ELSE cs.status
        END)::text AS sub_state,
       COALESCE(to_char(cs.ends_on, 'DD Mon YYYY'), '')::text AS ends_on_label
FROM members m
LEFT JOIN LATERAL (
  SELECT p.name AS plan_name, s.status, s.ends_on
  FROM subscriptions s
  JOIN plans p ON p.id = s.plan_id
  WHERE s.member_id = m.id AND s.status <> 'cancelled'
  ORDER BY (s.status IN ('active', 'frozen', 'pending')) DESC, s.ends_on DESC
  LIMIT 1
) cs ON true
WHERE (sqlc.arg(search)::text = ''
       OR m.full_name   ILIKE '%' || sqlc.arg(search)::text || '%'
       OR m.phone       ILIKE '%' || sqlc.arg(search)::text || '%'
       OR m.member_code ILIKE '%' || sqlc.arg(search)::text || '%')
  AND (
    sqlc.arg(filter)::text = 'all'
    OR (sqlc.arg(filter)::text = 'active'
        AND cs.status = 'active'
        AND cs.ends_on >= (now() AT TIME ZONE 'Africa/Lagos')::date)
    OR (sqlc.arg(filter)::text = 'expired'
        AND (cs.status = 'expired'
             OR (cs.status = 'active'
                 AND cs.ends_on < (now() AT TIME ZONE 'Africa/Lagos')::date)))
    OR (sqlc.arg(filter)::text = 'none' AND cs.status IS NULL)
  )
ORDER BY m.created_at DESC, m.id DESC
LIMIT sqlc.arg(page_size)::int OFFSET sqlc.arg(page_offset)::int;

-- name: GetMember :one
SELECT * FROM members WHERE id = $1;

-- name: CreateMember :one
INSERT INTO members (member_code, full_name, email, phone, date_of_birth,
                     emergency_contact_name, emergency_contact_phone, joined_on)
VALUES ('GYM-' || lpad(nextval('member_code_seq')::text, 5, '0'),
        $1, $2, $3, $4, $5, $6, $7)
RETURNING id;

-- name: UpdateMember :execrows
UPDATE members
SET full_name = $2, email = $3, phone = $4, date_of_birth = $5,
    emergency_contact_name = $6, emergency_contact_phone = $7, joined_on = $8
WHERE id = $1;

-- name: SetMemberActive :exec
UPDATE members SET is_active = $2 WHERE id = $1;
