set -a; source .env; set +a
goose -allow-missing -dir migrations postgres "$DATABASE_URL" up
