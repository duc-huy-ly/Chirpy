package tests

import (
	"testing"

	"github.com/duc-huy-ly/Chirpy/internal/auth"
)

func TestHashPassword(t *testing.T) {
	password := "secret"

	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}

	if hash == password {
		t.Fatal("password should be hashed")
	}
}

func TestCheckPasswordHash(t *testing.T) {
	password := "secret"

	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}

	ok, err := auth.CheckPasswordHash(password, hash)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected password to match")
	}
}
