package service

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

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

func TestValidateToken_MissingSub(t *testing.T) {
	auth := NewAuthService("test-secret")

	// Create a token with no "sub" claim
	claims := jwt.MapClaims{
		"role": "admin",
		"exp":  time.Now().Add(time.Hour).Unix(),
		"iat":  time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	_, err = auth.ValidateToken(tokenStr)
	if err == nil {
		t.Error("expected error for missing sub claim")
	}
}

func TestValidateToken_WrongTypeRole(t *testing.T) {
	auth := NewAuthService("test-secret")

	// Create a token with role as int instead of string
	claims := jwt.MapClaims{
		"sub":  "user-123",
		"role": 42,
		"exp":  time.Now().Add(time.Hour).Unix(),
		"iat":  time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	_, err = auth.ValidateToken(tokenStr)
	if err == nil {
		t.Error("expected error for non-string role claim")
	}
}
