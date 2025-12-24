package store

import (
	"context"
	"testing"
)

func TestStoreCloseNilSafe(t *testing.T) {
	var s *Store
	s.Close()

	s = &Store{}
	s.Close()
}

func TestOpenInvalidURL(t *testing.T) {
	if _, err := Open(context.Background(), "not-a-url"); err == nil {
		t.Fatal("expected error for invalid database url")
	}
}
