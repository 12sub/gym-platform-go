package services

import "testing"

func TestNewToken(t *testing.T) {
	a, ah := newToken()
	b, bh := newToken()
	if a == b || ah == bh {
		t.Fatal("two tokens should never match")
	}
	if len(a) != 43 {
		t.Errorf("token length = %d, want 43", len(a))
	}
	if hashToken(a) != ah {
		t.Error("hashing the same token twice should give the same result")
	}
	if ah == a {
		t.Error("the stored value must not be the token itself")
	}
}