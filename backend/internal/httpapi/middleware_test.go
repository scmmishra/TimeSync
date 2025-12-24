package httpapi

import (
	"net/http"
	"testing"
)

func TestKeyByDeviceID(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Device-Id", "device-123")

	key, err := keyByDeviceID(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key != "device:device-123" {
		t.Fatalf("unexpected key: %q", key)
	}
}

func TestKeyByDeviceIDMissing(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/", nil)

	if _, err := keyByDeviceID(req); err == nil {
		t.Fatal("expected error for missing device id")
	}
}
