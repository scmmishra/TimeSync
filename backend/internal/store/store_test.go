package store

import "testing"

func TestStoreCloseNilSafe(t *testing.T) {
	var s *Store
	s.Close()

	s = &Store{}
	s.Close()
}
