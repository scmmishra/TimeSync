package httpapi

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestNormalizeEmail(t *testing.T) {
	email, ok := normalizeEmail("  Foo@Example.com ")
	if !ok {
		t.Fatal("expected email to normalize")
	}
	if email != "foo@example.com" {
		t.Fatalf("expected normalized email, got %q", email)
	}

	if _, ok := normalizeEmail("not-an-email"); ok {
		t.Fatal("expected invalid email to fail")
	}
	if _, ok := normalizeEmail(" "); ok {
		t.Fatal("expected blank email to fail")
	}
}

func TestEmailDomain(t *testing.T) {
	domain, ok := emailDomain("user@example.com")
	if !ok || domain != "example.com" {
		t.Fatalf("expected domain example.com, got %q", domain)
	}
	if _, ok := emailDomain("missing-at"); ok {
		t.Fatal("expected invalid email to fail")
	}
	if _, ok := emailDomain("missing-domain@"); ok {
		t.Fatal("expected invalid email to fail")
	}
}

func TestNormalizeCode(t *testing.T) {
	if normalizeCode("  abc1234  ") != "ABC1234" {
		t.Fatal("expected normalizeCode to trim and uppercase")
	}
}

func TestIsValidCode(t *testing.T) {
	code, err := generateCode()
	if err != nil {
		t.Fatalf("generateCode error: %v", err)
	}
	if !isValidCode(code) {
		t.Fatalf("expected code to be valid: %q", code)
	}
	if isValidCode("abcd1234") {
		t.Fatal("expected lowercase to be invalid")
	}
	if isValidCode("INVALID!") {
		t.Fatal("expected non-alphabet characters to be invalid")
	}
}

func TestHashEqual(t *testing.T) {
	a := hashString("alpha")
	b := hashString("alpha")
	c := hashString("bravo")

	if !hashEqual(a, b) {
		t.Fatal("expected hashes to match")
	}
	if hashEqual(a, c) {
		t.Fatal("expected hashes to differ")
	}
	if hashEqual(a, []byte("short")) {
		t.Fatal("expected mismatched lengths to fail")
	}
}

func TestUUIDString(t *testing.T) {
	id := uuid.New()
	out := uuidString(pgtype.UUID{Bytes: id, Valid: true})
	if out != id.String() {
		t.Fatalf("expected %q, got %q", id.String(), out)
	}
	if uuidString(pgtype.UUID{}) != "" {
		t.Fatal("expected empty string for invalid uuid")
	}
}

func TestToTimestamptz(t *testing.T) {
	now := time.Now()
	ts := toTimestamptz(now)
	if !ts.Valid {
		t.Fatal("expected timestamptz to be valid")
	}
	if !ts.Time.Equal(now) {
		t.Fatalf("expected time %v, got %v", now, ts.Time)
	}
}
