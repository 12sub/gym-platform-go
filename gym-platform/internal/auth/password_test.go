package auth

import "testing"

func TestValidateNewPassword(t *testing.T) {
	bad := []struct{ pw, email string }{
		{"short", "a@b.com"},
		{"elevenchars", "a@b.com"},
		{"owner@gym.com", "owner@gym.com"},
		{"OWNER@GYM.COM", "owner@gym.com"},
	}
	for _, c := range bad {
		if err := ValidateNewPassword(c.pw, c.email); err == nil {
			t.Errorf("%q should be rejected", c.pw)
		}
	}
	if err := ValidateNewPassword("a long enough passphrase", "a@b.com"); err != nil {
		t.Errorf("a good passphrase was rejected: %v", err)
	}
}