package httpapi

import (
	"encoding/base64"
	"testing"
)

func TestGenerateToken(t *testing.T) {
	token, hash, err := generateToken()
	if err != nil {
		t.Fatalf("generateToken error: %v", err)
	}
	if token == "" {
		t.Fatal("expected token to be non-empty")
	}
	if len(hash) != 32 {
		t.Fatalf("expected sha256 hash length 32, got %d", len(hash))
	}
	decoded, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		t.Fatalf("token was not valid base64: %v", err)
	}
	if len(decoded) != tokenBytes {
		t.Fatalf("expected decoded length %d, got %d", tokenBytes, len(decoded))
	}
}

func TestGenerateCode(t *testing.T) {
	code, err := generateCode()
	if err != nil {
		t.Fatalf("generateCode error: %v", err)
	}
	if len(code) != codeLength {
		t.Fatalf("expected code length %d, got %d", codeLength, len(code))
	}
	for _, r := range code {
		if !stringsContainsRune(codeAlphabet, r) {
			t.Fatalf("unexpected rune in code: %q", r)
		}
	}
}

func stringsContainsRune(list []rune, r rune) bool {
	for _, item := range list {
		if item == r {
			return true
		}
	}
	return false
}
