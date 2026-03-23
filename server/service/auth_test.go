package service

import "testing"

func TestHashAndVerifyPassword(t *testing.T) {
	auth := NewAuthService("test-secret")

	hash, err := auth.HashPassword("mypassword")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}

	if !auth.VerifyPassword(hash, "mypassword") {
		t.Error("expected password to verify")
	}

	if auth.VerifyPassword(hash, "wrongpassword") {
		t.Error("expected wrong password to fail")
	}
}

func TestGenerateAndValidateToken(t *testing.T) {
	auth := NewAuthService("test-secret")

	token, err := auth.GenerateToken("user-123", "admin")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	claims, err := auth.ValidateToken(token)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}

	if claims.UserID != "user-123" {
		t.Errorf("expected user-123, got %s", claims.UserID)
	}
	if claims.Role != "admin" {
		t.Errorf("expected admin, got %s", claims.Role)
	}
}

func TestValidateToken_Invalid(t *testing.T) {
	auth := NewAuthService("test-secret")

	_, err := auth.ValidateToken("garbage")
	if err == nil {
		t.Error("expected error for invalid token")
	}
}

func TestValidateToken_WrongSecret(t *testing.T) {
	auth1 := NewAuthService("secret-1")
	auth2 := NewAuthService("secret-2")

	token, _ := auth1.GenerateToken("user-1", "user")
	_, err := auth2.ValidateToken(token)
	if err == nil {
		t.Error("expected error for wrong secret")
	}
}
