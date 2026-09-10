package tests

import (
	"net/http"
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

func TestGetBearerToken(t *testing.T) {
	tests := []struct {
		name      string
		headers   http.Header
		wantToken string
		wantErr   bool
	}{
		{
			name:    "no Authorization header",
			headers: make(http.Header),
			wantErr: true,
		},
		{
			name: "Authorization header without Bearer prefix",
			headers: http.Header{
				"Authorization": []string{"some-token"},
			},
			wantErr: true,
		},
		{
			name: "valid Bearer token",
			headers: http.Header{
				"Authorization": []string{"Bearer my-secret-token"},
			},
			wantToken: "my-secret-token",
			wantErr:   false,
		},
		{
			name: "Authorization header with invalid prefix",
			headers: http.Header{
				"Authorization": []string{"foobar whatever-token"},
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			token, err := auth.GetBearerToken(tc.headers)

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return // don't check token when we expect an error
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if token != tc.wantToken {
				t.Errorf("token = %q, want %q", token, tc.wantToken)
			}
		})
	}
}
