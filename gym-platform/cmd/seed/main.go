package main

import (
	"context"
	"log"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/12sub/gym-platform/internal/auth"
	"github.com/12sub/gym-platform/internal/db"
)

func main() {
	_ = godotenv.Load()

	email := strings.ToLower(strings.TrimSpace(os.Getenv("OWNER_EMAIL")))
	password := strings.TrimRight(os.Getenv("OWNER_PASSWORD"), "\r\n")
	if email == "" || len(password) < 12 {
		log.Fatal("set OWNER_EMAIL and an OWNER_PASSWORD of at least 12 characters in .env")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	q := db.New(pool)

	role, err := q.GetRoleBySlug(ctx, "owner")
	if err != nil {
		log.Fatalf("owner role missing (did the migrations run?): %v", err)
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		log.Fatalf("hash: %v", err)
	}

	id, err := q.UpsertUserByEmail(ctx, db.UpsertUserByEmailParams{
		FullName:     "Gym Owner",
		Email:        email,
		PasswordHash: hash,
		RoleID:       role.ID,
	})
	if err != nil {
		log.Fatalf("create owner: %v", err)
	}
	log.Printf("owner ready: id=%d email=%s", id, email)
}
