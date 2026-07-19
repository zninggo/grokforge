package auth

import "testing"

func TestTokenIssueAndParse(t *testing.T) {
	svc, err := NewTokenService("0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	pair, err := svc.Issue(1, "admin")
	if err != nil {
		t.Fatal(err)
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatal("empty tokens")
	}
	ac, err := svc.ParseAccess(pair.AccessToken)
	if err != nil {
		t.Fatal(err)
	}
	if ac.UserID != 1 || ac.Username != "admin" || ac.Kind != "access" {
		t.Fatalf("claims=%+v", ac)
	}
	rc, err := svc.ParseRefresh(pair.RefreshToken)
	if err != nil {
		t.Fatal(err)
	}
	if rc.Kind != "refresh" {
		t.Fatalf("kind=%s", rc.Kind)
	}
	if _, err := svc.ParseAccess(pair.RefreshToken); err == nil {
		t.Fatal("refresh must not parse as access")
	}
}

func TestTokenServiceRequiresSecret(t *testing.T) {
	if _, err := NewTokenService("short"); err == nil {
		t.Fatal("expected error")
	}
}
