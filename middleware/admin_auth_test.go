package middleware

import "testing"

func TestAdminTokenValidation(t *testing.T) {
	t.Setenv("SESSION_SECRET", "test-session-secret-with-more-than-32-characters")

	token := NewAdminToken()
	if !IsAdminToken(token) {
		t.Fatal("expected newly created admin token to be valid")
	}
	if IsAdminToken(token + "tampered") {
		t.Fatal("expected modified admin token to be invalid")
	}
	if IsAdminToken("") {
		t.Fatal("expected empty admin token to be invalid")
	}
}
