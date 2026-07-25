package service

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestHashPassword(t *testing.T) {
	hash, err := HashPassword("GravityLink!2026")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if hash == "GravityLink!2026" {
		t.Fatal("password was stored as plain text")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte("GravityLink!2026")); err != nil {
		t.Fatalf("hash does not verify: %v", err)
	}
}

func TestHashPasswordRejectsShortValue(t *testing.T) {
	if _, err := HashPassword("short"); err == nil {
		t.Fatal("expected short password to be rejected")
	}
}
