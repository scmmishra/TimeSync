package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"timesync/backend/internal/mailer"
	"timesync/backend/internal/sqlc"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type stubStore struct {
	querier sqlc.Querier
}

func (s *stubStore) BeginTx(ctx context.Context, opts pgx.TxOptions) (pgx.Tx, error) {
	return nil, errors.New("BeginTx not implemented")
}

func (s *stubStore) Querier() sqlc.Querier {
	return s.querier
}

func (s *stubStore) WithTx(tx pgx.Tx) sqlc.Querier {
	return s.querier
}

type stubMailer struct {
	calls     int
	lastEmail string
	lastCode  string
	err       error
}

func (m *stubMailer) SendVerificationCode(_ context.Context, email, code string) error {
	m.calls++
	m.lastEmail = email
	m.lastCode = code
	return m.err
}

type stubQuerier struct {
	createEmailVerificationCodeFn func(context.Context, sqlc.CreateEmailVerificationCodeParams) (sqlc.EmailVerificationCode, error)
	getAuthSessionByRefreshHashFn func(context.Context, sqlc.GetAuthSessionByRefreshHashParams) (sqlc.AuthSession, error)
	rotateAuthSessionFn           func(context.Context, sqlc.RotateAuthSessionParams) error
	createAuthSessionFn           func(context.Context, sqlc.CreateAuthSessionParams) (sqlc.AuthSession, error)
	markAuthSessionUsedFn         func(context.Context, sqlc.MarkAuthSessionUsedParams) error
	revokeAuthSessionFn           func(context.Context, sqlc.RevokeAuthSessionParams) error
}

func (s stubQuerier) CountTeamMembers(context.Context, pgtype.UUID) (int64, error) {
	panic("unexpected CountTeamMembers")
}

func (s stubQuerier) CreateAuthSession(ctx context.Context, arg sqlc.CreateAuthSessionParams) (sqlc.AuthSession, error) {
	if s.createAuthSessionFn == nil {
		panic("unexpected CreateAuthSession")
	}
	return s.createAuthSessionFn(ctx, arg)
}

func (s stubQuerier) CreateEmailVerificationCode(ctx context.Context, arg sqlc.CreateEmailVerificationCodeParams) (sqlc.EmailVerificationCode, error) {
	if s.createEmailVerificationCodeFn == nil {
		panic("unexpected CreateEmailVerificationCode")
	}
	return s.createEmailVerificationCodeFn(ctx, arg)
}

func (s stubQuerier) CreateTeam(context.Context, sqlc.CreateTeamParams) (sqlc.Team, error) {
	panic("unexpected CreateTeam")
}

func (s stubQuerier) CreateTeamMembership(context.Context, sqlc.CreateTeamMembershipParams) error {
	panic("unexpected CreateTeamMembership")
}

func (s stubQuerier) CreateUser(context.Context, sqlc.CreateUserParams) (sqlc.User, error) {
	panic("unexpected CreateUser")
}

func (s stubQuerier) GetAuthSessionByAccessHash(context.Context, sqlc.GetAuthSessionByAccessHashParams) (sqlc.AuthSession, error) {
	panic("unexpected GetAuthSessionByAccessHash")
}

func (s stubQuerier) GetAuthSessionByRefreshHash(ctx context.Context, arg sqlc.GetAuthSessionByRefreshHashParams) (sqlc.AuthSession, error) {
	if s.getAuthSessionByRefreshHashFn == nil {
		panic("unexpected GetAuthSessionByRefreshHash")
	}
	return s.getAuthSessionByRefreshHashFn(ctx, arg)
}

func (s stubQuerier) GetEmailVerificationCode(context.Context, sqlc.GetEmailVerificationCodeParams) (sqlc.EmailVerificationCode, error) {
	panic("unexpected GetEmailVerificationCode")
}

func (s stubQuerier) GetTeamByDomain(context.Context, string) (sqlc.Team, error) {
	panic("unexpected GetTeamByDomain")
}

func (s stubQuerier) GetTeamMembership(context.Context, sqlc.GetTeamMembershipParams) (sqlc.TeamMembership, error) {
	panic("unexpected GetTeamMembership")
}

func (s stubQuerier) GetUserByEmail(context.Context, string) (sqlc.User, error) {
	panic("unexpected GetUserByEmail")
}

func (s stubQuerier) GetUserByID(context.Context, pgtype.UUID) (sqlc.User, error) {
	panic("unexpected GetUserByID")
}

func (s stubQuerier) MarkAuthSessionUsed(ctx context.Context, arg sqlc.MarkAuthSessionUsedParams) error {
	if s.markAuthSessionUsedFn == nil {
		panic("unexpected MarkAuthSessionUsed")
	}
	return s.markAuthSessionUsedFn(ctx, arg)
}

func (s stubQuerier) MarkEmailVerificationCodeUsed(context.Context, sqlc.MarkEmailVerificationCodeUsedParams) error {
	panic("unexpected MarkEmailVerificationCodeUsed")
}

func (s stubQuerier) RevokeAuthSession(ctx context.Context, arg sqlc.RevokeAuthSessionParams) error {
	if s.revokeAuthSessionFn == nil {
		panic("unexpected RevokeAuthSession")
	}
	return s.revokeAuthSessionFn(ctx, arg)
}

func (s stubQuerier) RotateAuthSession(ctx context.Context, arg sqlc.RotateAuthSessionParams) error {
	if s.rotateAuthSessionFn == nil {
		panic("unexpected RotateAuthSession")
	}
	return s.rotateAuthSessionFn(ctx, arg)
}

func (s stubQuerier) UpdateUserVerifiedAt(context.Context, sqlc.UpdateUserVerifiedAtParams) (sqlc.User, error) {
	panic("unexpected UpdateUserVerifiedAt")
}

func TestHandleRequestCodeInvalidJSON(t *testing.T) {
	api := New(&stubStore{}, &mailer.LogMailer{}, Settings{}, nil)
	req := httptest.NewRequest(http.MethodPost, "/auth/request-code", bytes.NewBufferString("{bad json"))
	rec := httptest.NewRecorder()

	api.handleRequestCode(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestHandleRequestCodeSuccess(t *testing.T) {
	clock := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	q := stubQuerier{
		createEmailVerificationCodeFn: func(_ context.Context, arg sqlc.CreateEmailVerificationCodeParams) (sqlc.EmailVerificationCode, error) {
			if arg.Email != "user@example.com" {
				t.Fatalf("unexpected email: %q", arg.Email)
			}
			if arg.Code == "" {
				t.Fatal("expected code to be set")
			}
			if !arg.ExpiresAt.Valid || !arg.ExpiresAt.Time.Equal(clock.Add(10*time.Minute)) {
				t.Fatalf("unexpected expires at: %v", arg.ExpiresAt.Time)
			}
			return sqlc.EmailVerificationCode{}, nil
		},
	}
	m := &stubMailer{}
	api := New(&stubStore{querier: q}, m, Settings{
		CodeTTL:                10 * time.Minute,
		RequestCodeEmailLimit:  3,
		RequestCodeEmailWindow: time.Minute,
	}, nil)
	api.clock = func() time.Time { return clock }

	body, _ := json.Marshal(requestCodeRequest{Email: "user@example.com"})
	req := httptest.NewRequest(http.MethodPost, "/auth/request-code", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	api.handleRequestCode(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", rec.Code)
	}
	if m.calls != 1 {
		t.Fatalf("expected mailer to be called once, got %d", m.calls)
	}
}

func TestHandleRefreshSuccess(t *testing.T) {
	deviceID := "device-123"
	refreshToken := "refresh-token"
	now := time.Now()
	var rotated bool
	var created bool
	var marked bool

	q := stubQuerier{
		getAuthSessionByRefreshHashFn: func(_ context.Context, _ sqlc.GetAuthSessionByRefreshHashParams) (sqlc.AuthSession, error) {
			return sqlc.AuthSession{
				ID:               pgtype.UUID{Bytes: [16]byte{1}, Valid: true},
				UserID:           pgtype.UUID{Bytes: [16]byte{2}, Valid: true},
				DeviceIDHash:     hashString(deviceID),
				RefreshTokenHash: hashString(refreshToken),
			}, nil
		},
		rotateAuthSessionFn: func(context.Context, sqlc.RotateAuthSessionParams) error {
			rotated = true
			return nil
		},
		createAuthSessionFn: func(context.Context, sqlc.CreateAuthSessionParams) (sqlc.AuthSession, error) {
			created = true
			return sqlc.AuthSession{}, nil
		},
		markAuthSessionUsedFn: func(context.Context, sqlc.MarkAuthSessionUsedParams) error {
			marked = true
			return nil
		},
	}

	api := New(&stubStore{querier: q}, &mailer.LogMailer{}, Settings{
		AccessTTL:    15 * time.Minute,
		RefreshTTL:   24 * time.Hour,
		RefreshGrace: 30 * time.Second,
	}, nil)
	api.clock = func() time.Time { return now }

	body, _ := json.Marshal(refreshRequest{RefreshToken: refreshToken})
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader(body))
	req.Header.Set("X-Device-Id", deviceID)
	rec := httptest.NewRecorder()

	api.handleRefresh(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if !rotated || !created || !marked {
		t.Fatalf("expected rotate/create/mark to be called: rotated=%v created=%v marked=%v", rotated, created, marked)
	}
}

func TestHandleRefreshInvalidDevice(t *testing.T) {
	deviceID := "device-123"
	refreshToken := "refresh-token"

	q := stubQuerier{
		getAuthSessionByRefreshHashFn: func(_ context.Context, _ sqlc.GetAuthSessionByRefreshHashParams) (sqlc.AuthSession, error) {
			return sqlc.AuthSession{
				DeviceIDHash: hashString("other-device"),
			}, nil
		},
	}

	api := New(&stubStore{querier: q}, &mailer.LogMailer{}, Settings{
		AccessTTL:    15 * time.Minute,
		RefreshTTL:   24 * time.Hour,
		RefreshGrace: 30 * time.Second,
	}, nil)

	body, _ := json.Marshal(refreshRequest{RefreshToken: refreshToken})
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader(body))
	req.Header.Set("X-Device-Id", deviceID)
	rec := httptest.NewRecorder()

	api.handleRefresh(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
}

func TestHandleLogoutSuccess(t *testing.T) {
	deviceID := "device-123"
	refreshToken := "refresh-token"
	var revoked bool

	q := stubQuerier{
		getAuthSessionByRefreshHashFn: func(_ context.Context, _ sqlc.GetAuthSessionByRefreshHashParams) (sqlc.AuthSession, error) {
			return sqlc.AuthSession{
				ID:           pgtype.UUID{Bytes: [16]byte{1}, Valid: true},
				DeviceIDHash: hashString(deviceID),
			}, nil
		},
		revokeAuthSessionFn: func(context.Context, sqlc.RevokeAuthSessionParams) error {
			revoked = true
			return nil
		},
	}

	api := New(&stubStore{querier: q}, &mailer.LogMailer{}, Settings{}, nil)

	body, _ := json.Marshal(refreshRequest{RefreshToken: refreshToken})
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", bytes.NewReader(body))
	req.Header.Set("X-Device-Id", deviceID)
	rec := httptest.NewRecorder()

	api.handleLogout(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", rec.Code)
	}
	if !revoked {
		t.Fatal("expected revoke to be called")
	}
}
