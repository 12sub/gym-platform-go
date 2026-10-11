-- name: InsertAuditLog :exec
INSERT INTO audit_logs (user_id, action, entity_type, entity_id, metadata, ip_address)
VALUES (sqlc.narg(user_id), sqlc.arg(action), sqlc.narg(entity_type),
        sqlc.narg(entity_id), sqlc.narg(metadata), sqlc.narg(ip_address));