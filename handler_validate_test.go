package main

import "testing"

func TestCensorBadWords(t *testing.T) {
	result := censorBadWords([]string{"kerfuffle", "sharbert", "fornax"}, "This Is a kerfuffle situation")
	expected := "This Is a **** situation"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}
