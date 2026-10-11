package auth

import (
	"github.com/alexedwards/argon2id"
	"errors"
	"strings"
	"unicode/utf8"
)

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

var ErrWeakPassword = errors.New("Use at least 12 characters, and not your email address.")

// ValidateNewPassword is the one place the password rule lives.
func ValidateNewPassword(password, email string) error {
	n := utf8.RuneCountInString(password)
	if n < 12 || n > 200 || strings.EqualFold(password, email) {
		return ErrWeakPassword
	}
	return nil
}