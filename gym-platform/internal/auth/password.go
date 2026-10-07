package auth

import "github.com/alexedwards/argon2id"

func HashPassword(password string) (string, error) {
	return argon2id.CreateHash(password, argon2id.DefaultParams)
}

func CheckPassword(password, hash string) (bool, error) {
	return argon2id.ComparePasswordAndHash(password, hash)
}

// dummyHash lets us spend the same time on unknown emails as on real ones,
// so response timing doesn't reveal which emails exist.
var dummyHash, _ = argon2id.CreateHash("not-a-real-password", argon2id.DefaultParams)

func BurnTime(password string) {
	_, _ = argon2id.ComparePasswordAndHash(password, dummyHash)
}