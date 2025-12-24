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
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type stubStore struct {
	querier   sqlc.Querier
	beginTxFn func(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

func (s *stubStore) BeginTx(ctx context.Context, opts pgx.TxOptions) (pgx.Tx, error) {
	if s.beginTxFn == nil {
		return nil, errors.New("BeginTx not implemented")
	}
	return s.beginTxFn(ctx, opts)
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
	countTeamMembersFn            func(context.Context, pgtype.UUID) (int64, error)
	createAuthSessionFn           func(context.Context, sqlc.CreateAuthSessionParams) (sqlc.AuthSession, error)
	createEmailVerificationCodeFn func(context.Context, sqlc.CreateEmailVerificationCodeParams) (sqlc.EmailVerificationCode, error)
	createTeamFn                  func(context.Context, sqlc.CreateTeamParams) (sqlc.Team, error)
	createTeamMembershipFn        func(context.Context, sqlc.CreateTeamMembershipParams) error
	createUserFn                  func(context.Context, sqlc.CreateUserParams) (sqlc.User, error)
	getAuthSessionByRefreshHashFn func(context.Context, sqlc.GetAuthSessionByRefreshHashParams) (sqlc.AuthSession, error)
	getEmailVerificationCodeFn    func(context.Context, sqlc.GetEmailVerificationCodeParams) (sqlc.EmailVerificationCode, error)
	getTeamByDomainFn             func(context.Context, string) (sqlc.Team, error)
	getTeamMembershipFn           func(context.Context, sqlc.GetTeamMembershipParams) (sqlc.TeamMembership, error)
	getUserByEmailFn              func(context.Context, string) (sqlc.User, error)
	markAuthSessionUsedFn         func(context.Context, sqlc.MarkAuthSessionUsedParams) error
	markEmailVerificationCodeFn   func(context.Context, sqlc.MarkEmailVerificationCodeUsedParams) error
	revokeAuthSessionFn           func(context.Context, sqlc.RevokeAuthSessionParams) error
	rotateAuthSessionFn           func(context.Context, sqlc.RotateAuthSessionParams) error
	updateUserVerifiedAtFn        func(context.Context, sqlc.UpdateUserVerifiedAtParams) (sqlc.User, error)
}

func (s stubQuerier) CountTeamMembers(ctx context.Context, teamID pgtype.UUID) (int64, error) {
	if s.countTeamMembersFn == nil {
		panic("unexpected CountTeamMembers")
	}
	return s.countTeamMembersFn(ctx, teamID)
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

func (s stubQuerier) CreateTeam(ctx context.Context, arg sqlc.CreateTeamParams) (sqlc.Team, error) {
	if s.createTeamFn == nil {
		panic("unexpected CreateTeam")
	}
	return s.createTeamFn(ctx, arg)
}

func (s stubQuerier) CreateTeamMembership(ctx context.Context, arg sqlc.CreateTeamMembershipParams) error {
	if s.createTeamMembershipFn == nil {
		panic("unexpected CreateTeamMembership")
	}
	return s.createTeamMembershipFn(ctx, arg)
}

func (s stubQuerier) CreateUser(ctx context.Context, arg sqlc.CreateUserParams) (sqlc.User, error) {
	if s.createUserFn == nil {
		panic("unexpected CreateUser")
	}
	return s.createUserFn(ctx, arg)
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

func (s stubQuerier) GetEmailVerificationCode(ctx context.Context, arg sqlc.GetEmailVerificationCodeParams) (sqlc.EmailVerificationCode, error) {
	if s.getEmailVerificationCodeFn == nil {
		panic("unexpected GetEmailVerificationCode")
	}
	return s.getEmailVerificationCodeFn(ctx, arg)
}

func (s stubQuerier) GetTeamByDomain(ctx context.Context, domain string) (sqlc.Team, error) {
	if s.getTeamByDomainFn == nil {
		panic("unexpected GetTeamByDomain")
	}
	return s.getTeamByDomainFn(ctx, domain)
}

func (s stubQuerier) GetTeamMembership(ctx context.Context, arg sqlc.GetTeamMembershipParams) (sqlc.TeamMembership, error) {
	if s.getTeamMembershipFn == nil {
		panic("unexpected GetTeamMembership")
	}
	return s.getTeamMembershipFn(ctx, arg)
}

func (s stubQuerier) GetUserByEmail(ctx context.Context, email string) (sqlc.User, error) {
	if s.getUserByEmailFn == nil {
		panic("unexpected GetUserByEmail")
	}
	return s.getUserByEmailFn(ctx, email)
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

func (s stubQuerier) MarkEmailVerificationCodeUsed(ctx context.Context, arg sqlc.MarkEmailVerificationCodeUsedParams) error {
	if s.markEmailVerificationCodeFn == nil {
		panic("unexpected MarkEmailVerificationCodeUsed")
	}
	return s.markEmailVerificationCodeFn(ctx, arg)
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

func (s stubQuerier) UpdateUserVerifiedAt(ctx context.Context, arg sqlc.UpdateUserVerifiedAtParams) (sqlc.User, error) {
	if s.updateUserVerifiedAtFn == nil {
		panic("unexpected UpdateUserVerifiedAt")
	}
	return s.updateUserVerifiedAtFn(ctx, arg)
}

type testTx struct {
	committed bool
	rolled    bool
}

func (t *testTx) Begin(context.Context) (pgx.Tx, error) { return t, nil }
func (t *testTx) Commit(context.Context) error {
	t.committed = true
	return nil
}
func (t *testTx) Rollback(context.Context) error {
	t.rolled = true
	return nil
}
func (t *testTx) CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error) {
	return 0, nil
}
func (t *testTx) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults { return nil }
func (t *testTx) LargeObjects() pgx.LargeObjects                         { return pgx.LargeObjects{} }
func (t *testTx) Prepare(context.Context, string, string) (*pgconn.StatementDescription, error) {
	return nil, nil
}
func (t *testTx) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (t *testTx) Query(context.Context, string, ...any) (pgx.Rows, error) { return nil, nil }
func (t *testTx) QueryRow(context.Context, string, ...any) pgx.Row        { return nil }
func (t *testTx) Conn() *pgx.Conn                                         { return nil }

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

func TestHandleRequestCodeMailerFailure(t *testing.T) {
	q := stubQuerier{
		createEmailVerificationCodeFn: func(context.Context, sqlc.CreateEmailVerificationCodeParams) (sqlc.EmailVerificationCode, error) {
			return sqlc.EmailVerificationCode{}, nil
		},
	}
	m := &stubMailer{err: errors.New("smtp down")}
	api := New(&stubStore{querier: q}, m, Settings{
		CodeTTL:                10 * time.Minute,
		RequestCodeEmailLimit:  3,
		RequestCodeEmailWindow: time.Minute,
	}, nil)

	body, _ := json.Marshal(requestCodeRequest{Email: "user@example.com"})
	req := httptest.NewRequest(http.MethodPost, "/auth/request-code", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	api.handleRequestCode(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rec.Code)
	}
}

func TestHandleRequestCodeRateLimited(t *testing.T) {
	q := stubQuerier{
		createEmailVerificationCodeFn: func(context.Context, sqlc.CreateEmailVerificationCodeParams) (sqlc.EmailVerificationCode, error) {
			return sqlc.EmailVerificationCode{}, nil
		},
	}
	m := &stubMailer{}
	api := New(&stubStore{querier: q}, m, Settings{
		CodeTTL:                10 * time.Minute,
		RequestCodeEmailLimit:  1,
		RequestCodeEmailWindow: time.Hour,
	}, nil)

	body, _ := json.Marshal(requestCodeRequest{Email: "user@example.com"})
	req := httptest.NewRequest(http.MethodPost, "/auth/request-code", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	api.handleRequestCode(rec, req)

	req2 := httptest.NewRequest(http.MethodPost, "/auth/request-code", bytes.NewReader(body))
	rec2 := httptest.NewRecorder()
	api.handleRequestCode(rec2, req2)

	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status 429, got %d", rec2.Code)
	}
}

func TestHandleVerifyCodeSuccess(t *testing.T) {
	email := "user@example.com"
	code := "ABCD2345"
	deviceID := "device-123"
	now := time.Date(2024, 1, 2, 9, 0, 0, 0, time.UTC)
	tx := &testTx{}
	var membershipRole string

	q := stubQuerier{
		getEmailVerificationCodeFn: func(_ context.Context, arg sqlc.GetEmailVerificationCodeParams) (sqlc.EmailVerificationCode, error) {
			if arg.Email != email || arg.Code != code {
				t.Fatalf("unexpected email/code: %q/%q", arg.Email, arg.Code)
			}
			if !arg.ExpiresAt.Valid || !arg.ExpiresAt.Time.Equal(now) {
				t.Fatalf("unexpected expires at: %v", arg.ExpiresAt.Time)
			}
			return sqlc.EmailVerificationCode{ID: pgtype.UUID{Bytes: [16]byte{1}, Valid: true}}, nil
		},
		markEmailVerificationCodeFn: func(_ context.Context, arg sqlc.MarkEmailVerificationCodeUsedParams) error {
			if !arg.UsedAt.Valid || !arg.UsedAt.Time.Equal(now) {
				t.Fatalf("unexpected used at: %v", arg.UsedAt.Time)
			}
			return nil
		},
		getUserByEmailFn: func(context.Context, string) (sqlc.User, error) {
			return sqlc.User{}, pgx.ErrNoRows
		},
		createUserFn: func(_ context.Context, arg sqlc.CreateUserParams) (sqlc.User, error) {
			if arg.Email != email {
				t.Fatalf("unexpected email: %q", arg.Email)
			}
			return sqlc.User{ID: pgtype.UUID{Bytes: [16]byte{2}, Valid: true}, Email: arg.Email}, nil
		},
		getTeamByDomainFn: func(context.Context, string) (sqlc.Team, error) {
			return sqlc.Team{}, pgx.ErrNoRows
		},
		createTeamFn: func(_ context.Context, arg sqlc.CreateTeamParams) (sqlc.Team, error) {
			return sqlc.Team{ID: pgtype.UUID{Bytes: [16]byte{3}, Valid: true}, Domain: arg.Domain, Name: arg.Name}, nil
		},
		getTeamMembershipFn: func(context.Context, sqlc.GetTeamMembershipParams) (sqlc.TeamMembership, error) {
			return sqlc.TeamMembership{}, pgx.ErrNoRows
		},
		countTeamMembersFn: func(context.Context, pgtype.UUID) (int64, error) {
			return 0, nil
		},
		createTeamMembershipFn: func(_ context.Context, arg sqlc.CreateTeamMembershipParams) error {
			membershipRole = arg.Role
			return nil
		},
		createAuthSessionFn: func(_ context.Context, arg sqlc.CreateAuthSessionParams) (sqlc.AuthSession, error) {
			if !arg.AccessExpiresAt.Valid || !arg.RefreshExpiresAt.Valid {
				t.Fatal("expected expires to be set")
			}
			return sqlc.AuthSession{}, nil
		},
	}

	api := New(&stubStore{
		querier: q,
		beginTxFn: func(context.Context, pgx.TxOptions) (pgx.Tx, error) {
			return tx, nil
		},
	}, &mailer.LogMailer{}, Settings{
		AccessTTL:              15 * time.Minute,
		RefreshTTL:             24 * time.Hour,
		CodeTTL:                10 * time.Minute,
		VerifyCodeEmailLimit:   5,
		VerifyCodeEmailWindow:  15 * time.Minute,
		VerifyCodeLock:         15 * time.Minute,
		TeamSizeLimit:          30,
		RequestCodeEmailLimit:  1,
		RequestCodeEmailWindow: time.Minute,
	}, nil)
	api.clock = func() time.Time { return now }

	body, _ := json.Marshal(verifyCodeRequest{Email: email, Code: code})
	req := httptest.NewRequest(http.MethodPost, "/auth/verify-code", bytes.NewReader(body))
	req.Header.Set("X-Device-Id", deviceID)
	rec := httptest.NewRecorder()

	api.handleVerifyCode(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if membershipRole != "admin" {
		t.Fatalf("expected admin role, got %q", membershipRole)
	}
	if !tx.committed {
		t.Fatal("expected tx to be committed")
	}
}

func TestHandleVerifyCodeInvalidCode(t *testing.T) {
	email := "user@example.com"
	code := "ABCD2345"
	deviceID := "device-123"
	now := time.Now()

	q := stubQuerier{
		getEmailVerificationCodeFn: func(context.Context, sqlc.GetEmailVerificationCodeParams) (sqlc.EmailVerificationCode, error) {
			return sqlc.EmailVerificationCode{}, pgx.ErrNoRows
		},
	}

	api := New(&stubStore{
		querier: q,
		beginTxFn: func(context.Context, pgx.TxOptions) (pgx.Tx, error) {
			return &testTx{}, nil
		},
	}, &mailer.LogMailer{}, Settings{
		VerifyCodeEmailLimit:  5,
		VerifyCodeEmailWindow: 15 * time.Minute,
		VerifyCodeLock:        15 * time.Minute,
	}, nil)
	api.clock = func() time.Time { return now }

	body, _ := json.Marshal(verifyCodeRequest{Email: email, Code: code})
	req := httptest.NewRequest(http.MethodPost, "/auth/verify-code", bytes.NewReader(body))
	req.Header.Set("X-Device-Id", deviceID)
	rec := httptest.NewRecorder()

	api.handleVerifyCode(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
}

func TestHandleVerifyCodeLockedAfterFailures(t *testing.T) {
	email := "user@example.com"
	code := "ABCD2345"
	deviceID := "device-123"
	now := time.Now()

	q := stubQuerier{
		getEmailVerificationCodeFn: func(context.Context, sqlc.GetEmailVerificationCodeParams) (sqlc.EmailVerificationCode, error) {
			return sqlc.EmailVerificationCode{}, pgx.ErrNoRows
		},
	}

	api := New(&stubStore{
		querier: q,
		beginTxFn: func(context.Context, pgx.TxOptions) (pgx.Tx, error) {
			return &testTx{}, nil
		},
	}, &mailer.LogMailer{}, Settings{
		VerifyCodeEmailLimit:  1,
		VerifyCodeEmailWindow: 15 * time.Minute,
		VerifyCodeLock:        15 * time.Minute,
	}, nil)
	api.clock = func() time.Time { return now }

	body, _ := json.Marshal(verifyCodeRequest{Email: email, Code: code})
	req := httptest.NewRequest(http.MethodPost, "/auth/verify-code", bytes.NewReader(body))
	req.Header.Set("X-Device-Id", deviceID)
	rec := httptest.NewRecorder()

	api.handleVerifyCode(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status 429, got %d", rec.Code)
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

func TestHandleRefreshWithinGraceDoesNotRotate(t *testing.T) {
	deviceID := "device-123"
	refreshToken := "refresh-token"
	now := time.Now()
	var rotated bool

	q := stubQuerier{
		getAuthSessionByRefreshHashFn: func(_ context.Context, _ sqlc.GetAuthSessionByRefreshHashParams) (sqlc.AuthSession, error) {
			return sqlc.AuthSession{
				ID:               pgtype.UUID{Bytes: [16]byte{1}, Valid: true},
				UserID:           pgtype.UUID{Bytes: [16]byte{2}, Valid: true},
				DeviceIDHash:     hashString(deviceID),
				RefreshTokenHash: hashString(refreshToken),
				RotatedAt:        pgtype.Timestamptz{Time: now.Add(-10 * time.Second), Valid: true},
			}, nil
		},
		rotateAuthSessionFn: func(context.Context, sqlc.RotateAuthSessionParams) error {
			rotated = true
			return nil
		},
		createAuthSessionFn: func(context.Context, sqlc.CreateAuthSessionParams) (sqlc.AuthSession, error) {
			return sqlc.AuthSession{}, nil
		},
		markAuthSessionUsedFn: func(context.Context, sqlc.MarkAuthSessionUsedParams) error {
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
	if rotated {
		t.Fatal("expected not to rotate within grace period")
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

func TestHandleLogoutInvalidDevice(t *testing.T) {
	deviceID := "device-123"
	refreshToken := "refresh-token"

	q := stubQuerier{
		getAuthSessionByRefreshHashFn: func(_ context.Context, _ sqlc.GetAuthSessionByRefreshHashParams) (sqlc.AuthSession, error) {
			return sqlc.AuthSession{
				DeviceIDHash: hashString("other-device"),
			}, nil
		},
	}

	api := New(&stubStore{querier: q}, &mailer.LogMailer{}, Settings{}, nil)

	body, _ := json.Marshal(refreshRequest{RefreshToken: refreshToken})
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", bytes.NewReader(body))
	req.Header.Set("X-Device-Id", deviceID)
	rec := httptest.NewRecorder()

	api.handleLogout(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
}

func TestHandleVerifyCodeMissingDeviceID(t *testing.T) {
	api := New(&stubStore{}, &mailer.LogMailer{}, Settings{}, nil)
	body, _ := json.Marshal(verifyCodeRequest{Email: "user@example.com", Code: "ABCD2345"})
	req := httptest.NewRequest(http.MethodPost, "/auth/verify-code", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	api.handleVerifyCode(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestHandleVerifyCodeInvalidEmail(t *testing.T) {
	api := New(&stubStore{}, &mailer.LogMailer{}, Settings{}, nil)
	body, _ := json.Marshal(verifyCodeRequest{Email: "bad-email", Code: "ABCD2345"})
	req := httptest.NewRequest(http.MethodPost, "/auth/verify-code", bytes.NewReader(body))
	req.Header.Set("X-Device-Id", "device-123")
	rec := httptest.NewRecorder()

	api.handleVerifyCode(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestHandleVerifyCodeInvalidFormat(t *testing.T) {
	api := New(&stubStore{}, &mailer.LogMailer{}, Settings{}, nil)
	body, _ := json.Marshal(verifyCodeRequest{Email: "user@example.com", Code: "ABCD230!"})
	req := httptest.NewRequest(http.MethodPost, "/auth/verify-code", bytes.NewReader(body))
	req.Header.Set("X-Device-Id", "device-123")
	rec := httptest.NewRecorder()

	api.handleVerifyCode(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestHandleRequestCodeInvalidEmail(t *testing.T) {
	api := New(&stubStore{}, &mailer.LogMailer{}, Settings{}, nil)
	body, _ := json.Marshal(requestCodeRequest{Email: "bad-email"})
	req := httptest.NewRequest(http.MethodPost, "/auth/request-code", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	api.handleRequestCode(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestHandleRequestCodeDBFailure(t *testing.T) {
	q := stubQuerier{
		createEmailVerificationCodeFn: func(context.Context, sqlc.CreateEmailVerificationCodeParams) (sqlc.EmailVerificationCode, error) {
			return sqlc.EmailVerificationCode{}, errors.New("insert failed")
		},
	}
	api := New(&stubStore{querier: q}, &mailer.LogMailer{}, Settings{
		CodeTTL:                10 * time.Minute,
		RequestCodeEmailLimit:  3,
		RequestCodeEmailWindow: time.Minute,
	}, nil)

	body, _ := json.Marshal(requestCodeRequest{Email: "user@example.com"})
	req := httptest.NewRequest(http.MethodPost, "/auth/request-code", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	api.handleRequestCode(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rec.Code)
	}
}

func TestHandleVerifyCodeInvalidBody(t *testing.T) {
	api := New(&stubStore{}, &mailer.LogMailer{}, Settings{}, nil)
	req := httptest.NewRequest(http.MethodPost, "/auth/verify-code", bytes.NewBufferString("{bad json"))
	rec := httptest.NewRecorder()

	api.handleVerifyCode(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestHandleVerifyCodeBeginTxFailure(t *testing.T) {
	api := New(&stubStore{
		beginTxFn: func(context.Context, pgx.TxOptions) (pgx.Tx, error) {
			return nil, errors.New("begin failed")
		},
	}, &mailer.LogMailer{}, Settings{}, nil)

	body, _ := json.Marshal(verifyCodeRequest{Email: "user@example.com", Code: "ABCD2345"})
	req := httptest.NewRequest(http.MethodPost, "/auth/verify-code", bytes.NewReader(body))
	req.Header.Set("X-Device-Id", "device-123")
	rec := httptest.NewRecorder()

	api.handleVerifyCode(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rec.Code)
	}
}

func TestHandleVerifyCodeLookupFailure(t *testing.T) {
	q := stubQuerier{
		getEmailVerificationCodeFn: func(context.Context, sqlc.GetEmailVerificationCodeParams) (sqlc.EmailVerificationCode, error) {
			return sqlc.EmailVerificationCode{}, errors.New("db failed")
		},
	}

	api := New(&stubStore{
		querier: q,
		beginTxFn: func(context.Context, pgx.TxOptions) (pgx.Tx, error) {
			return &testTx{}, nil
		},
	}, &mailer.LogMailer{}, Settings{
		VerifyCodeEmailLimit:  5,
		VerifyCodeEmailWindow: 15 * time.Minute,
		VerifyCodeLock:        15 * time.Minute,
	}, nil)

	body, _ := json.Marshal(verifyCodeRequest{Email: "user@example.com", Code: "ABCD2345"})
	req := httptest.NewRequest(http.MethodPost, "/auth/verify-code", bytes.NewReader(body))
	req.Header.Set("X-Device-Id", "device-123")
	rec := httptest.NewRecorder()

	api.handleVerifyCode(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rec.Code)
	}
}

func TestHandleVerifyCodeGetUserFailure(t *testing.T) {
	q := stubQuerier{
		getEmailVerificationCodeFn: func(context.Context, sqlc.GetEmailVerificationCodeParams) (sqlc.EmailVerificationCode, error) {
			return sqlc.EmailVerificationCode{ID: pgtype.UUID{Bytes: [16]byte{1}, Valid: true}}, nil
		},
		markEmailVerificationCodeFn: func(context.Context, sqlc.MarkEmailVerificationCodeUsedParams) error {
			return nil
		},
		getUserByEmailFn: func(context.Context, string) (sqlc.User, error) {
			return sqlc.User{}, errors.New("db failed")
		},
	}

	api := New(&stubStore{
		querier: q,
		beginTxFn: func(context.Context, pgx.TxOptions) (pgx.Tx, error) {
			return &testTx{}, nil
		},
	}, &mailer.LogMailer{}, Settings{
		VerifyCodeEmailLimit:  5,
		VerifyCodeEmailWindow: 15 * time.Minute,
		VerifyCodeLock:        15 * time.Minute,
	}, nil)

	body, _ := json.Marshal(verifyCodeRequest{Email: "user@example.com", Code: "ABCD2345"})
	req := httptest.NewRequest(http.MethodPost, "/auth/verify-code", bytes.NewReader(body))
	req.Header.Set("X-Device-Id", "device-123")
	rec := httptest.NewRecorder()

	api.handleVerifyCode(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rec.Code)
	}
}

func TestHandleVerifyCodeUpdateVerifiedFailure(t *testing.T) {
	email := "user@example.com"
	now := time.Now()

	q := stubQuerier{
		getEmailVerificationCodeFn: func(context.Context, sqlc.GetEmailVerificationCodeParams) (sqlc.EmailVerificationCode, error) {
			return sqlc.EmailVerificationCode{ID: pgtype.UUID{Bytes: [16]byte{1}, Valid: true}}, nil
		},
		markEmailVerificationCodeFn: func(context.Context, sqlc.MarkEmailVerificationCodeUsedParams) error {
			return nil
		},
		getUserByEmailFn: func(context.Context, string) (sqlc.User, error) {
			return sqlc.User{ID: pgtype.UUID{Bytes: [16]byte{2}, Valid: true}, Email: email}, nil
		},
		updateUserVerifiedAtFn: func(context.Context, sqlc.UpdateUserVerifiedAtParams) (sqlc.User, error) {
			return sqlc.User{}, errors.New("update failed")
		},
	}

	api := New(&stubStore{
		querier: q,
		beginTxFn: func(context.Context, pgx.TxOptions) (pgx.Tx, error) {
			return &testTx{}, nil
		},
	}, &mailer.LogMailer{}, Settings{
		VerifyCodeEmailLimit:  5,
		VerifyCodeEmailWindow: 15 * time.Minute,
		VerifyCodeLock:        15 * time.Minute,
	}, nil)
	api.clock = func() time.Time { return now }

	body, _ := json.Marshal(verifyCodeRequest{Email: email, Code: "ABCD2345"})
	req := httptest.NewRequest(http.MethodPost, "/auth/verify-code", bytes.NewReader(body))
	req.Header.Set("X-Device-Id", "device-123")
	rec := httptest.NewRecorder()

	api.handleVerifyCode(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rec.Code)
	}
}

func TestHandleVerifyCodeGetTeamFailure(t *testing.T) {
	email := "user@example.com"

	q := stubQuerier{
		getEmailVerificationCodeFn: func(context.Context, sqlc.GetEmailVerificationCodeParams) (sqlc.EmailVerificationCode, error) {
			return sqlc.EmailVerificationCode{ID: pgtype.UUID{Bytes: [16]byte{1}, Valid: true}}, nil
		},
		markEmailVerificationCodeFn: func(context.Context, sqlc.MarkEmailVerificationCodeUsedParams) error {
			return nil
		},
		getUserByEmailFn: func(context.Context, string) (sqlc.User, error) {
			return sqlc.User{ID: pgtype.UUID{Bytes: [16]byte{2}, Valid: true}, Email: email, EmailVerifiedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true}}, nil
		},
		getTeamByDomainFn: func(context.Context, string) (sqlc.Team, error) {
			return sqlc.Team{}, errors.New("db failed")
		},
	}

	api := New(&stubStore{
		querier: q,
		beginTxFn: func(context.Context, pgx.TxOptions) (pgx.Tx, error) {
			return &testTx{}, nil
		},
	}, &mailer.LogMailer{}, Settings{
		VerifyCodeEmailLimit:  5,
		VerifyCodeEmailWindow: 15 * time.Minute,
		VerifyCodeLock:        15 * time.Minute,
	}, nil)

	body, _ := json.Marshal(verifyCodeRequest{Email: email, Code: "ABCD2345"})
	req := httptest.NewRequest(http.MethodPost, "/auth/verify-code", bytes.NewReader(body))
	req.Header.Set("X-Device-Id", "device-123")
	rec := httptest.NewRecorder()

	api.handleVerifyCode(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rec.Code)
	}
}

func TestHandleVerifyCodeCountMembersFailure(t *testing.T) {
	email := "user@example.com"

	q := stubQuerier{
		getEmailVerificationCodeFn: func(context.Context, sqlc.GetEmailVerificationCodeParams) (sqlc.EmailVerificationCode, error) {
			return sqlc.EmailVerificationCode{ID: pgtype.UUID{Bytes: [16]byte{1}, Valid: true}}, nil
		},
		markEmailVerificationCodeFn: func(context.Context, sqlc.MarkEmailVerificationCodeUsedParams) error {
			return nil
		},
		getUserByEmailFn: func(context.Context, string) (sqlc.User, error) {
			return sqlc.User{ID: pgtype.UUID{Bytes: [16]byte{2}, Valid: true}, Email: email, EmailVerifiedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true}}, nil
		},
		getTeamByDomainFn: func(context.Context, string) (sqlc.Team, error) {
			return sqlc.Team{ID: pgtype.UUID{Bytes: [16]byte{3}, Valid: true}, Domain: "example.com", Name: "example.com"}, nil
		},
		getTeamMembershipFn: func(context.Context, sqlc.GetTeamMembershipParams) (sqlc.TeamMembership, error) {
			return sqlc.TeamMembership{}, pgx.ErrNoRows
		},
		countTeamMembersFn: func(context.Context, pgtype.UUID) (int64, error) {
			return 0, errors.New("count failed")
		},
	}

	api := New(&stubStore{
		querier: q,
		beginTxFn: func(context.Context, pgx.TxOptions) (pgx.Tx, error) {
			return &testTx{}, nil
		},
	}, &mailer.LogMailer{}, Settings{
		VerifyCodeEmailLimit:  5,
		VerifyCodeEmailWindow: 15 * time.Minute,
		VerifyCodeLock:        15 * time.Minute,
	}, nil)

	body, _ := json.Marshal(verifyCodeRequest{Email: email, Code: "ABCD2345"})
	req := httptest.NewRequest(http.MethodPost, "/auth/verify-code", bytes.NewReader(body))
	req.Header.Set("X-Device-Id", "device-123")
	rec := httptest.NewRecorder()

	api.handleVerifyCode(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rec.Code)
	}
}

func TestHandleVerifyCodeCreateMembershipFailure(t *testing.T) {
	email := "user@example.com"

	q := stubQuerier{
		getEmailVerificationCodeFn: func(context.Context, sqlc.GetEmailVerificationCodeParams) (sqlc.EmailVerificationCode, error) {
			return sqlc.EmailVerificationCode{ID: pgtype.UUID{Bytes: [16]byte{1}, Valid: true}}, nil
		},
		markEmailVerificationCodeFn: func(context.Context, sqlc.MarkEmailVerificationCodeUsedParams) error {
			return nil
		},
		getUserByEmailFn: func(context.Context, string) (sqlc.User, error) {
			return sqlc.User{ID: pgtype.UUID{Bytes: [16]byte{2}, Valid: true}, Email: email, EmailVerifiedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true}}, nil
		},
		getTeamByDomainFn: func(context.Context, string) (sqlc.Team, error) {
			return sqlc.Team{ID: pgtype.UUID{Bytes: [16]byte{3}, Valid: true}, Domain: "example.com", Name: "example.com"}, nil
		},
		getTeamMembershipFn: func(context.Context, sqlc.GetTeamMembershipParams) (sqlc.TeamMembership, error) {
			return sqlc.TeamMembership{}, pgx.ErrNoRows
		},
		countTeamMembersFn: func(context.Context, pgtype.UUID) (int64, error) {
			return 0, nil
		},
		createTeamMembershipFn: func(context.Context, sqlc.CreateTeamMembershipParams) error {
			return errors.New("insert failed")
		},
	}

	api := New(&stubStore{
		querier: q,
		beginTxFn: func(context.Context, pgx.TxOptions) (pgx.Tx, error) {
			return &testTx{}, nil
		},
	}, &mailer.LogMailer{}, Settings{
		VerifyCodeEmailLimit:  5,
		VerifyCodeEmailWindow: 15 * time.Minute,
		VerifyCodeLock:        15 * time.Minute,
		TeamSizeLimit:         30,
	}, nil)

	body, _ := json.Marshal(verifyCodeRequest{Email: email, Code: "ABCD2345"})
	req := httptest.NewRequest(http.MethodPost, "/auth/verify-code", bytes.NewReader(body))
	req.Header.Set("X-Device-Id", "device-123")
	rec := httptest.NewRecorder()

	api.handleVerifyCode(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rec.Code)
	}
}

func TestHandleRefreshMissingToken(t *testing.T) {
	api := New(&stubStore{}, &mailer.LogMailer{}, Settings{}, nil)
	body, _ := json.Marshal(refreshRequest{RefreshToken: ""})
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader(body))
	req.Header.Set("X-Device-Id", "device-123")
	rec := httptest.NewRecorder()

	api.handleRefresh(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestHandleRefreshMissingDevice(t *testing.T) {
	api := New(&stubStore{}, &mailer.LogMailer{}, Settings{}, nil)
	body, _ := json.Marshal(refreshRequest{RefreshToken: "refresh-token"})
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	api.handleRefresh(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestHandleRefreshInvalidBody(t *testing.T) {
	api := New(&stubStore{}, &mailer.LogMailer{}, Settings{}, nil)
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewBufferString("{bad json"))
	rec := httptest.NewRecorder()

	api.handleRefresh(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestHandleLogoutMissingToken(t *testing.T) {
	api := New(&stubStore{}, &mailer.LogMailer{}, Settings{}, nil)
	body, _ := json.Marshal(refreshRequest{RefreshToken: ""})
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", bytes.NewReader(body))
	req.Header.Set("X-Device-Id", "device-123")
	rec := httptest.NewRecorder()

	api.handleLogout(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestHandleLogoutMissingDevice(t *testing.T) {
	api := New(&stubStore{}, &mailer.LogMailer{}, Settings{}, nil)
	body, _ := json.Marshal(refreshRequest{RefreshToken: "refresh-token"})
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	api.handleLogout(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestHandleVerifyCodePreLocked(t *testing.T) {
	api := New(&stubStore{}, &mailer.LogMailer{}, Settings{}, nil)
	now := time.Now()
	api.failLimit.RegisterFailure("user@example.com", 1, time.Minute, time.Minute, now)

	body, _ := json.Marshal(verifyCodeRequest{Email: "user@example.com", Code: "ABCD2345"})
	req := httptest.NewRequest(http.MethodPost, "/auth/verify-code", bytes.NewReader(body))
	req.Header.Set("X-Device-Id", "device-123")
	rec := httptest.NewRecorder()

	api.handleVerifyCode(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status 429, got %d", rec.Code)
	}
}

func TestHandleVerifyCodeTeamFull(t *testing.T) {
	email := "user@example.com"
	code := "ABCD2345"
	deviceID := "device-123"
	now := time.Now()

	q := stubQuerier{
		getEmailVerificationCodeFn: func(context.Context, sqlc.GetEmailVerificationCodeParams) (sqlc.EmailVerificationCode, error) {
			return sqlc.EmailVerificationCode{ID: pgtype.UUID{Bytes: [16]byte{1}, Valid: true}}, nil
		},
		markEmailVerificationCodeFn: func(context.Context, sqlc.MarkEmailVerificationCodeUsedParams) error {
			return nil
		},
		getUserByEmailFn: func(context.Context, string) (sqlc.User, error) {
			return sqlc.User{
				ID:              pgtype.UUID{Bytes: [16]byte{2}, Valid: true},
				Email:           email,
				EmailVerifiedAt: pgtype.Timestamptz{Time: now, Valid: true},
			}, nil
		},
		getTeamByDomainFn: func(context.Context, string) (sqlc.Team, error) {
			return sqlc.Team{ID: pgtype.UUID{Bytes: [16]byte{3}, Valid: true}, Domain: "example.com", Name: "example.com"}, nil
		},
		getTeamMembershipFn: func(context.Context, sqlc.GetTeamMembershipParams) (sqlc.TeamMembership, error) {
			return sqlc.TeamMembership{}, pgx.ErrNoRows
		},
		countTeamMembersFn: func(context.Context, pgtype.UUID) (int64, error) {
			return 1, nil
		},
	}

	api := New(&stubStore{
		querier: q,
		beginTxFn: func(context.Context, pgx.TxOptions) (pgx.Tx, error) {
			return &testTx{}, nil
		},
	}, &mailer.LogMailer{}, Settings{
		VerifyCodeEmailLimit:  5,
		VerifyCodeEmailWindow: 15 * time.Minute,
		VerifyCodeLock:        15 * time.Minute,
		TeamSizeLimit:         1,
	}, nil)
	api.clock = func() time.Time { return now }

	body, _ := json.Marshal(verifyCodeRequest{Email: email, Code: code})
	req := httptest.NewRequest(http.MethodPost, "/auth/verify-code", bytes.NewReader(body))
	req.Header.Set("X-Device-Id", deviceID)
	rec := httptest.NewRecorder()

	api.handleVerifyCode(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", rec.Code)
	}
}

func TestHandleVerifyCodeMarkUsedFailure(t *testing.T) {
	email := "user@example.com"
	code := "ABCD2345"

	q := stubQuerier{
		getEmailVerificationCodeFn: func(context.Context, sqlc.GetEmailVerificationCodeParams) (sqlc.EmailVerificationCode, error) {
			return sqlc.EmailVerificationCode{ID: pgtype.UUID{Bytes: [16]byte{1}, Valid: true}}, nil
		},
		markEmailVerificationCodeFn: func(context.Context, sqlc.MarkEmailVerificationCodeUsedParams) error {
			return errors.New("update failed")
		},
	}

	api := New(&stubStore{
		querier: q,
		beginTxFn: func(context.Context, pgx.TxOptions) (pgx.Tx, error) {
			return &testTx{}, nil
		},
	}, &mailer.LogMailer{}, Settings{
		VerifyCodeEmailLimit:  5,
		VerifyCodeEmailWindow: 15 * time.Minute,
		VerifyCodeLock:        15 * time.Minute,
	}, nil)

	body, _ := json.Marshal(verifyCodeRequest{Email: email, Code: code})
	req := httptest.NewRequest(http.MethodPost, "/auth/verify-code", bytes.NewReader(body))
	req.Header.Set("X-Device-Id", "device-123")
	rec := httptest.NewRecorder()

	api.handleVerifyCode(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rec.Code)
	}
}

func TestHandleVerifyCodeCreateAuthSessionFailure(t *testing.T) {
	email := "user@example.com"
	code := "ABCD2345"
	deviceID := "device-123"
	now := time.Now()

	q := stubQuerier{
		getEmailVerificationCodeFn: func(context.Context, sqlc.GetEmailVerificationCodeParams) (sqlc.EmailVerificationCode, error) {
			return sqlc.EmailVerificationCode{ID: pgtype.UUID{Bytes: [16]byte{1}, Valid: true}}, nil
		},
		markEmailVerificationCodeFn: func(context.Context, sqlc.MarkEmailVerificationCodeUsedParams) error {
			return nil
		},
		getUserByEmailFn: func(context.Context, string) (sqlc.User, error) {
			return sqlc.User{ID: pgtype.UUID{Bytes: [16]byte{2}, Valid: true}, Email: email, EmailVerifiedAt: pgtype.Timestamptz{Time: now, Valid: true}}, nil
		},
		getTeamByDomainFn: func(context.Context, string) (sqlc.Team, error) {
			return sqlc.Team{ID: pgtype.UUID{Bytes: [16]byte{3}, Valid: true}, Domain: "example.com", Name: "example.com"}, nil
		},
		getTeamMembershipFn: func(context.Context, sqlc.GetTeamMembershipParams) (sqlc.TeamMembership, error) {
			return sqlc.TeamMembership{}, pgx.ErrNoRows
		},
		countTeamMembersFn: func(context.Context, pgtype.UUID) (int64, error) {
			return 0, nil
		},
		createTeamMembershipFn: func(context.Context, sqlc.CreateTeamMembershipParams) error {
			return nil
		},
		createAuthSessionFn: func(context.Context, sqlc.CreateAuthSessionParams) (sqlc.AuthSession, error) {
			return sqlc.AuthSession{}, errors.New("insert failed")
		},
	}

	api := New(&stubStore{
		querier: q,
		beginTxFn: func(context.Context, pgx.TxOptions) (pgx.Tx, error) {
			return &testTx{}, nil
		},
	}, &mailer.LogMailer{}, Settings{
		AccessTTL:              15 * time.Minute,
		RefreshTTL:             24 * time.Hour,
		CodeTTL:                10 * time.Minute,
		VerifyCodeEmailLimit:   5,
		VerifyCodeEmailWindow:  15 * time.Minute,
		VerifyCodeLock:         15 * time.Minute,
		TeamSizeLimit:          30,
		RequestCodeEmailLimit:  1,
		RequestCodeEmailWindow: time.Minute,
	}, nil)
	api.clock = func() time.Time { return now }

	body, _ := json.Marshal(verifyCodeRequest{Email: email, Code: code})
	req := httptest.NewRequest(http.MethodPost, "/auth/verify-code", bytes.NewReader(body))
	req.Header.Set("X-Device-Id", deviceID)
	rec := httptest.NewRecorder()

	api.handleVerifyCode(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rec.Code)
	}
}

func TestHandleRefreshRotatedExpired(t *testing.T) {
	deviceID := "device-123"
	refreshToken := "refresh-token"
	now := time.Now()

	q := stubQuerier{
		getAuthSessionByRefreshHashFn: func(_ context.Context, _ sqlc.GetAuthSessionByRefreshHashParams) (sqlc.AuthSession, error) {
			return sqlc.AuthSession{
				DeviceIDHash:     hashString(deviceID),
				RefreshTokenHash: hashString(refreshToken),
				RotatedAt:        pgtype.Timestamptz{Time: now.Add(-time.Minute), Valid: true},
			}, nil
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

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
}

func TestHandleRefreshNotFound(t *testing.T) {
	q := stubQuerier{
		getAuthSessionByRefreshHashFn: func(context.Context, sqlc.GetAuthSessionByRefreshHashParams) (sqlc.AuthSession, error) {
			return sqlc.AuthSession{}, pgx.ErrNoRows
		},
	}

	api := New(&stubStore{querier: q}, &mailer.LogMailer{}, Settings{}, nil)

	body, _ := json.Marshal(refreshRequest{RefreshToken: "refresh-token"})
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader(body))
	req.Header.Set("X-Device-Id", "device-123")
	rec := httptest.NewRecorder()

	api.handleRefresh(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
}

func TestHandleRefreshRotateFailure(t *testing.T) {
	deviceID := "device-123"
	refreshToken := "refresh-token"

	q := stubQuerier{
		getAuthSessionByRefreshHashFn: func(_ context.Context, _ sqlc.GetAuthSessionByRefreshHashParams) (sqlc.AuthSession, error) {
			return sqlc.AuthSession{
				ID:           pgtype.UUID{Bytes: [16]byte{1}, Valid: true},
				DeviceIDHash: hashString(deviceID),
			}, nil
		},
		rotateAuthSessionFn: func(context.Context, sqlc.RotateAuthSessionParams) error {
			return errors.New("rotate failed")
		},
	}

	api := New(&stubStore{querier: q}, &mailer.LogMailer{}, Settings{
		RefreshGrace: 30 * time.Second,
	}, nil)

	body, _ := json.Marshal(refreshRequest{RefreshToken: refreshToken})
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader(body))
	req.Header.Set("X-Device-Id", deviceID)
	rec := httptest.NewRecorder()

	api.handleRefresh(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rec.Code)
	}
}

func TestHandleRefreshCreateSessionFailure(t *testing.T) {
	deviceID := "device-123"
	refreshToken := "refresh-token"

	q := stubQuerier{
		getAuthSessionByRefreshHashFn: func(_ context.Context, _ sqlc.GetAuthSessionByRefreshHashParams) (sqlc.AuthSession, error) {
			return sqlc.AuthSession{
				ID:           pgtype.UUID{Bytes: [16]byte{1}, Valid: true},
				UserID:       pgtype.UUID{Bytes: [16]byte{2}, Valid: true},
				DeviceIDHash: hashString(deviceID),
			}, nil
		},
		rotateAuthSessionFn: func(context.Context, sqlc.RotateAuthSessionParams) error {
			return nil
		},
		createAuthSessionFn: func(context.Context, sqlc.CreateAuthSessionParams) (sqlc.AuthSession, error) {
			return sqlc.AuthSession{}, errors.New("insert failed")
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

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rec.Code)
	}
}

func TestHandleLogoutRevokeFailure(t *testing.T) {
	deviceID := "device-123"
	refreshToken := "refresh-token"

	q := stubQuerier{
		getAuthSessionByRefreshHashFn: func(_ context.Context, _ sqlc.GetAuthSessionByRefreshHashParams) (sqlc.AuthSession, error) {
			return sqlc.AuthSession{
				ID:           pgtype.UUID{Bytes: [16]byte{1}, Valid: true},
				DeviceIDHash: hashString(deviceID),
			}, nil
		},
		revokeAuthSessionFn: func(context.Context, sqlc.RevokeAuthSessionParams) error {
			return errors.New("revoke failed")
		},
	}

	api := New(&stubStore{querier: q}, &mailer.LogMailer{}, Settings{}, nil)

	body, _ := json.Marshal(refreshRequest{RefreshToken: refreshToken})
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", bytes.NewReader(body))
	req.Header.Set("X-Device-Id", deviceID)
	rec := httptest.NewRecorder()

	api.handleLogout(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rec.Code)
	}
}
