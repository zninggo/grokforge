package auth

import "testing"

func TestHashAndCheckPassword(t *testing.T) {
	hash, err := HashPassword("correct-horse")
	if err != nil {
		t.Fatal(err)
	}
	if hash == "" || hash == "correct-horse" {
		t.Fatalf("unexpected hash %q", hash)
	}
	if !CheckPassword(hash, "correct-horse") {
		t.Fatal("expected match")
	}
	if CheckPassword(hash, "wrong-password") {
		t.Fatal("expected mismatch")
	}
	if CheckPassword("", "correct-horse") {
		t.Fatal("empty hash must fail")
	}
}

func TestValidatePassword(t *testing.T) {
	if err := ValidatePassword("short"); err == nil {
		t.Fatal("expected short password error")
	}
	if err := ValidatePassword("longenough"); err != nil {
		t.Fatal(err)
	}
}

func TestHashPasswordRejectsShort(t *testing.T) {
	if _, err := HashPassword("1234567"); err == nil {
		t.Fatal("expected error")
	}
}
