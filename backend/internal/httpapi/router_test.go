package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"timesync/backend/internal/mailer"
)

func TestHandlerHealth(t *testing.T) {
	settings := Settings{
		RequestCodeIPLimit:     1,
		RequestCodeIPWindow:    time.Minute,
		VerifyCodeIPLimit:      1,
		VerifyCodeIPWindow:     time.Minute,
		RefreshDeviceLimit:     1,
		RefreshDeviceWindow:    time.Minute,
		RequestCodeEmailLimit:  1,
		RequestCodeEmailWindow: time.Minute,
		VerifyCodeEmailLimit:   1,
		VerifyCodeEmailWindow:  time.Minute,
		VerifyCodeLock:         time.Minute,
	}
	api := New(nil, &mailer.LogMailer{}, settings, nil)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	api.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if body := rec.Body.String(); body != "ok" {
		t.Fatalf("unexpected body: %q", body)
	}
}
