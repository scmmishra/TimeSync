package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	payload := map[string]string{"ok": "true"}

	writeJSON(rec, http.StatusCreated, payload)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected content-type application/json, got %q", ct)
	}

	var decoded map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&decoded); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if decoded["ok"] != "true" {
		t.Fatalf("unexpected payload: %#v", decoded)
	}
}

func TestWriteError(t *testing.T) {
	rec := httptest.NewRecorder()

	writeError(rec, http.StatusBadRequest, "bad input")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	var resp errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.Error != "bad input" {
		t.Fatalf("unexpected error message: %q", resp.Error)
	}
}
