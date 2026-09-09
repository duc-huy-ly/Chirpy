package tests

import (
	"testing"
	"time"

	"github.com/duc-huy-ly/Chirpy/internal/auth"
	"github.com/google/uuid"
)

func TestMakeJWTAndValidateJWT(t *testing.T) {
	userID := uuid.New()
	secret := "test-secret"

	token, err := auth.MakeJWT(userID, secret, time.Hour)
	if err != nil {
		t.Fatalf("MakeJWT() returned an error: %v", err)
	}

	gotUserID, err := auth.ValidateJWT(token, secret)
	if err != nil {
		t.Fatalf("ValidateJWT() returned an error: %v", err)
	}
	if gotUserID != userID {
		t.Fatalf("ValidateJWT() returned user ID %q, want %q", gotUserID, userID)
	}
}

func TestValidateJWTRejectsWrongSecret(t *testing.T) {
	token, err := auth.MakeJWT(uuid.New(), "correct-secret", time.Hour)
	if err != nil {
		t.Fatalf("MakeJWT() returned an error: %v", err)
	}

	if _, err := auth.ValidateJWT(token, "wrong-secret"); err == nil {
		t.Fatal("ValidateJWT() succeeded with the wrong secret")
	}
}

func TestValidateJWTRejectsMalformedToken(t *testing.T) {
	if _, err := auth.ValidateJWT("not-a-jwt", "test-secret"); err == nil {
		t.Fatal("ValidateJWT() succeeded with a malformed token")
	}
}
