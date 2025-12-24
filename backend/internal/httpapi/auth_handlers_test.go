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

// stubQuerier builder for cleaner test setup
type querierBuilder struct {
	fns map[string]interface{}
}

func newQuerierBuilder() *querierBuilder {
	return &querierBuilder{fns: make(map[string]interface{})}
}

func (b *querierBuilder) onCountTeamMembers(fn func(context.Context, pgtype.UUID) (int64, error)) *querierBuilder {
	b.fns["countTeamMembers"] = fn
	return b
}

func (b *querierBuilder) onCreateAuthSession(fn func(context.Context, sqlc.CreateAuthSessionParams) (sqlc.AuthSession, error)) *querierBuilder {
	b.fns["createAuthSession"] = fn
	return b
}

func (b *querierBuilder) onCreateEmailVerificationCode(fn func(context.Context, sqlc.CreateEmailVerificationCodeParams) (sqlc.EmailVerificationCode, error)) *querierBuilder {
	b.fns["createEmailVerificationCode"] = fn
	return b
}

func (b *querierBuilder) onCreateTeam(fn func(context.Context, sqlc.CreateTeamParams) (sqlc.Team, error)) *querierBuilder {
	b.fns["createTeam"] = fn
	return b
}

func (b *querierBuilder) onCreateTeamMembership(fn func(context.Context, sqlc.CreateTeamMembershipParams) error) *querierBuilder {
	b.fns["createTeamMembership"] = fn
	return b
}

func (b *querierBuilder) onCreateUser(fn func(context.Context, sqlc.CreateUserParams) (sqlc.User, error)) *querierBuilder {
	b.fns["createUser"] = fn
	return b
}

func (b *querierBuilder) onGetAuthSessionByRefreshHash(fn func(context.Context, sqlc.GetAuthSessionByRefreshHashParams) (sqlc.AuthSession, error)) *querierBuilder {
	b.fns["getAuthSessionByRefreshHash"] = fn
	return b
}

func (b *querierBuilder) onGetEmailVerificationCode(fn func(context.Context, sqlc.GetEmailVerificationCodeParams) (sqlc.EmailVerificationCode, error)) *querierBuilder {
	b.fns["getEmailVerificationCode"] = fn
	return b
}

func (b *querierBuilder) onGetTeamByDomain(fn func(context.Context, string) (sqlc.Team, error)) *querierBuilder {
	b.fns["getTeamByDomain"] = fn
	return b
}

func (b *querierBuilder) onGetTeamMembership(fn func(context.Context, sqlc.GetTeamMembershipParams) (sqlc.TeamMembership, error)) *querierBuilder {
	b.fns["getTeamMembership"] = fn
	return b
}

func (b *querierBuilder) onGetUserByEmail(fn func(context.Context, string) (sqlc.User, error)) *querierBuilder {
	b.fns["getUserByEmail"] = fn
	return b
}

func (b *querierBuilder) onMarkAuthSessionUsed(fn func(context.Context, sqlc.MarkAuthSessionUsedParams) error) *querierBuilder {
	b.fns["markAuthSessionUsed"] = fn
	return b
}

func (b *querierBuilder) onMarkEmailVerificationCodeUsed(fn func(context.Context, sqlc.MarkEmailVerificationCodeUsedParams) error) *querierBuilder {
	b.fns["markEmailVerificationCodeUsed"] = fn
	return b
}

func (b *querierBuilder) onRevokeAuthSession(fn func(context.Context, sqlc.RevokeAuthSessionParams) error) *querierBuilder {
	b.fns["revokeAuthSession"] = fn
	return b
}

func (b *querierBuilder) onRotateAuthSession(fn func(context.Context, sqlc.RotateAuthSessionParams) error) *querierBuilder {
	b.fns["rotateAuthSession"] = fn
	return b
}

func (b *querierBuilder) onUpdateUserVerifiedAt(fn func(context.Context, sqlc.UpdateUserVerifiedAtParams) (sqlc.User, error)) *querierBuilder {
	b.fns["updateUserVerifiedAt"] = fn
	return b
}

func (b *querierBuilder) build() sqlc.Querier {
	return &builtQuerier{fns: b.fns}
}

type builtQuerier struct {
	fns map[string]interface{}
}

func (q *builtQuerier) CountTeamMembers(ctx context.Context, teamID pgtype.UUID) (int64, error) {
	if fn, ok := q.fns["countTeamMembers"]; ok {
		return fn.(func(context.Context, pgtype.UUID) (int64, error))(ctx, teamID)
	}
	return 0, nil
}

func (q *builtQuerier) CreateAuthSession(ctx context.Context, arg sqlc.CreateAuthSessionParams) (sqlc.AuthSession, error) {
	if fn, ok := q.fns["createAuthSession"]; ok {
		return fn.(func(context.Context, sqlc.CreateAuthSessionParams) (sqlc.AuthSession, error))(ctx, arg)
	}
	return sqlc.AuthSession{}, nil
}

func (q *builtQuerier) CreateEmailVerificationCode(ctx context.Context, arg sqlc.CreateEmailVerificationCodeParams) (sqlc.EmailVerificationCode, error) {
	if fn, ok := q.fns["createEmailVerificationCode"]; ok {
		return fn.(func(context.Context, sqlc.CreateEmailVerificationCodeParams) (sqlc.EmailVerificationCode, error))(ctx, arg)
	}
	return sqlc.EmailVerificationCode{}, nil
}

func (q *builtQuerier) CreateTeam(ctx context.Context, arg sqlc.CreateTeamParams) (sqlc.Team, error) {
	if fn, ok := q.fns["createTeam"]; ok {
		return fn.(func(context.Context, sqlc.CreateTeamParams) (sqlc.Team, error))(ctx, arg)
	}
	return sqlc.Team{}, nil
}

func (q *builtQuerier) CreateTeamMembership(ctx context.Context, arg sqlc.CreateTeamMembershipParams) error {
	if fn, ok := q.fns["createTeamMembership"]; ok {
		return fn.(func(context.Context, sqlc.CreateTeamMembershipParams) error)(ctx, arg)
	}
	return nil
}

func (q *builtQuerier) CreateUser(ctx context.Context, arg sqlc.CreateUserParams) (sqlc.User, error) {
	if fn, ok := q.fns["createUser"]; ok {
		return fn.(func(context.Context, sqlc.CreateUserParams) (sqlc.User, error))(ctx, arg)
	}
	return sqlc.User{}, nil
}

func (q *builtQuerier) GetAuthSessionByAccessHash(context.Context, sqlc.GetAuthSessionByAccessHashParams) (sqlc.AuthSession, error) {
	panic("unexpected GetAuthSessionByAccessHash")
}

func (q *builtQuerier) GetAuthSessionByRefreshHash(ctx context.Context, arg sqlc.GetAuthSessionByRefreshHashParams) (sqlc.AuthSession, error) {
	if fn, ok := q.fns["getAuthSessionByRefreshHash"]; ok {
		return fn.(func(context.Context, sqlc.GetAuthSessionByRefreshHashParams) (sqlc.AuthSession, error))(ctx, arg)
	}
	return sqlc.AuthSession{}, nil
}

func (q *builtQuerier) GetEmailVerificationCode(ctx context.Context, arg sqlc.GetEmailVerificationCodeParams) (sqlc.EmailVerificationCode, error) {
	if fn, ok := q.fns["getEmailVerificationCode"]; ok {
		return fn.(func(context.Context, sqlc.GetEmailVerificationCodeParams) (sqlc.EmailVerificationCode, error))(ctx, arg)
	}
	return sqlc.EmailVerificationCode{}, nil
}

func (q *builtQuerier) GetTeamByDomain(ctx context.Context, domain string) (sqlc.Team, error) {
	if fn, ok := q.fns["getTeamByDomain"]; ok {
		return fn.(func(context.Context, string) (sqlc.Team, error))(ctx, domain)
	}
	return sqlc.Team{}, nil
}

func (q *builtQuerier) GetTeamMembership(ctx context.Context, arg sqlc.GetTeamMembershipParams) (sqlc.TeamMembership, error) {
	if fn, ok := q.fns["getTeamMembership"]; ok {
		return fn.(func(context.Context, sqlc.GetTeamMembershipParams) (sqlc.TeamMembership, error))(ctx, arg)
	}
	return sqlc.TeamMembership{}, nil
}

func (q *builtQuerier) GetUserByEmail(ctx context.Context, email string) (sqlc.User, error) {
	if fn, ok := q.fns["getUserByEmail"]; ok {
		return fn.(func(context.Context, string) (sqlc.User, error))(ctx, email)
	}
	return sqlc.User{}, nil
}

func (q *builtQuerier) GetUserByID(context.Context, pgtype.UUID) (sqlc.User, error) {
	panic("unexpected GetUserByID")
}

func (q *builtQuerier) MarkAuthSessionUsed(ctx context.Context, arg sqlc.MarkAuthSessionUsedParams) error {
	if fn, ok := q.fns["markAuthSessionUsed"]; ok {
		return fn.(func(context.Context, sqlc.MarkAuthSessionUsedParams) error)(ctx, arg)
	}
	return nil
}

func (q *builtQuerier) MarkEmailVerificationCodeUsed(ctx context.Context, arg sqlc.MarkEmailVerificationCodeUsedParams) error {
	if fn, ok := q.fns["markEmailVerificationCodeUsed"]; ok {
		return fn.(func(context.Context, sqlc.MarkEmailVerificationCodeUsedParams) error)(ctx, arg)
	}
	return nil
}

func (q *builtQuerier) RevokeAuthSession(ctx context.Context, arg sqlc.RevokeAuthSessionParams) error {
	if fn, ok := q.fns["revokeAuthSession"]; ok {
		return fn.(func(context.Context, sqlc.RevokeAuthSessionParams) error)(ctx, arg)
	}
	return nil
}

func (q *builtQuerier) RotateAuthSession(ctx context.Context, arg sqlc.RotateAuthSessionParams) error {
	if fn, ok := q.fns["rotateAuthSession"]; ok {
		return fn.(func(context.Context, sqlc.RotateAuthSessionParams) error)(ctx, arg)
	}
	return nil
}

func (q *builtQuerier) UpdateUserVerifiedAt(ctx context.Context, arg sqlc.UpdateUserVerifiedAtParams) (sqlc.User, error) {
	if fn, ok := q.fns["updateUserVerifiedAt"]; ok {
		return fn.(func(context.Context, sqlc.UpdateUserVerifiedAtParams) (sqlc.User, error))(ctx, arg)
	}
	return sqlc.User{}, nil
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

// Test input validation with table-driven approach
func TestHandleRequestCodeInputValidation(t *testing.T) {
	tests := []struct {
		name string
		body []byte
		code int
	}{
		{"invalid JSON", []byte("{bad json"), http.StatusBadRequest},
		{"invalid email", func() []byte { b, _ := json.Marshal(requestCodeRequest{Email: "bad-email"}); return b }(), http.StatusBadRequest},
		{"empty email", func() []byte { b, _ := json.Marshal(requestCodeRequest{Email: ""}); return b }(), http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := New(&stubStore{}, &mailer.LogMailer{}, Settings{}, nil)
			req := httptest.NewRequest(http.MethodPost, "/auth/request-code", bytes.NewReader(tt.body))
			rec := httptest.NewRecorder()

			api.handleRequestCode(rec, req)

			if rec.Code != tt.code {
				t.Fatalf("expected status %d, got %d", tt.code, rec.Code)
			}
		})
	}
}

func TestHandleRequestCode(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		clock := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
		q := newQuerierBuilder().
			onCreateEmailVerificationCode(func(_ context.Context, arg sqlc.CreateEmailVerificationCodeParams) (sqlc.EmailVerificationCode, error) {
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
			}).
			build()

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
	})

	t.Run("mailer failure", func(t *testing.T) {
		q := newQuerierBuilder().
			onCreateEmailVerificationCode(func(context.Context, sqlc.CreateEmailVerificationCodeParams) (sqlc.EmailVerificationCode, error) {
				return sqlc.EmailVerificationCode{}, nil
			}).
			build()

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
	})

	t.Run("rate limited", func(t *testing.T) {
		q := newQuerierBuilder().
			onCreateEmailVerificationCode(func(context.Context, sqlc.CreateEmailVerificationCodeParams) (sqlc.EmailVerificationCode, error) {
				return sqlc.EmailVerificationCode{}, nil
			}).
			build()

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
	})

	t.Run("db failure", func(t *testing.T) {
		q := newQuerierBuilder().
			onCreateEmailVerificationCode(func(context.Context, sqlc.CreateEmailVerificationCodeParams) (sqlc.EmailVerificationCode, error) {
				return sqlc.EmailVerificationCode{}, errors.New("insert failed")
			}).
			build()

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
	})
}

func TestHandleVerifyCodeInputValidation(t *testing.T) {
	tests := []struct {
		name      string
		email     string
		code      string
		deviceID  string
		expectErr bool
	}{
		{"invalid email", "bad-email", "ABCD2345", "device-123", true},
		{"missing device", "user@example.com", "ABCD2345", "", true},
		{"invalid code format", "user@example.com", "ABCD230!", "device-123", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := New(&stubStore{
				beginTxFn: func(context.Context, pgx.TxOptions) (pgx.Tx, error) {
					return &testTx{}, nil
				},
			}, &mailer.LogMailer{}, Settings{}, nil)

			body, _ := json.Marshal(verifyCodeRequest{Email: tt.email, Code: tt.code})
			req := httptest.NewRequest(http.MethodPost, "/auth/verify-code", bytes.NewReader(body))
			if tt.deviceID != "" {
				req.Header.Set("X-Device-Id", tt.deviceID)
			}
			rec := httptest.NewRecorder()

			api.handleVerifyCode(rec, req)

			if rec.Code < 400 {
				t.Fatalf("expected error status, got %d", rec.Code)
			}
		})
	}
}

func TestHandleVerifyCode(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		email := "user@example.com"
		code := "ABCD2345"
		deviceID := "device-123"
		now := time.Date(2024, 1, 2, 9, 0, 0, 0, time.UTC)
		tx := &testTx{}
		var membershipRole string

		q := newQuerierBuilder().
			onGetEmailVerificationCode(func(_ context.Context, arg sqlc.GetEmailVerificationCodeParams) (sqlc.EmailVerificationCode, error) {
				if arg.Email != email || arg.Code != code {
					t.Fatalf("unexpected email/code: %q/%q", arg.Email, arg.Code)
				}
				if !arg.ExpiresAt.Valid || !arg.ExpiresAt.Time.Equal(now) {
					t.Fatalf("unexpected expires at: %v", arg.ExpiresAt.Time)
				}
				return sqlc.EmailVerificationCode{ID: pgtype.UUID{Bytes: [16]byte{1}, Valid: true}}, nil
			}).
			onMarkEmailVerificationCodeUsed(func(_ context.Context, arg sqlc.MarkEmailVerificationCodeUsedParams) error {
				if !arg.UsedAt.Valid || !arg.UsedAt.Time.Equal(now) {
					t.Fatalf("unexpected used at: %v", arg.UsedAt.Time)
				}
				return nil
			}).
			onGetUserByEmail(func(context.Context, string) (sqlc.User, error) {
				return sqlc.User{}, pgx.ErrNoRows
			}).
			onCreateUser(func(_ context.Context, arg sqlc.CreateUserParams) (sqlc.User, error) {
				if arg.Email != email {
					t.Fatalf("unexpected email: %q", arg.Email)
				}
				return sqlc.User{ID: pgtype.UUID{Bytes: [16]byte{2}, Valid: true}, Email: arg.Email}, nil
			}).
			onGetTeamByDomain(func(context.Context, string) (sqlc.Team, error) {
				return sqlc.Team{}, pgx.ErrNoRows
			}).
			onCreateTeam(func(_ context.Context, arg sqlc.CreateTeamParams) (sqlc.Team, error) {
				return sqlc.Team{ID: pgtype.UUID{Bytes: [16]byte{3}, Valid: true}, Domain: arg.Domain, Name: arg.Name}, nil
			}).
			onGetTeamMembership(func(context.Context, sqlc.GetTeamMembershipParams) (sqlc.TeamMembership, error) {
				return sqlc.TeamMembership{}, pgx.ErrNoRows
			}).
			onCountTeamMembers(func(context.Context, pgtype.UUID) (int64, error) {
				return 0, nil
			}).
			onCreateTeamMembership(func(_ context.Context, arg sqlc.CreateTeamMembershipParams) error {
				membershipRole = arg.Role
				return nil
			}).
			onCreateAuthSession(func(_ context.Context, arg sqlc.CreateAuthSessionParams) (sqlc.AuthSession, error) {
				if !arg.AccessExpiresAt.Valid || !arg.RefreshExpiresAt.Valid {
					t.Fatal("expected expires to be set")
				}
				return sqlc.AuthSession{}, nil
			}).
			build()

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
	})

	t.Run("invalid code", func(t *testing.T) {
		q := newQuerierBuilder().
			onGetEmailVerificationCode(func(context.Context, sqlc.GetEmailVerificationCodeParams) (sqlc.EmailVerificationCode, error) {
				return sqlc.EmailVerificationCode{}, pgx.ErrNoRows
			}).
			build()

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

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", rec.Code)
		}
	})

	t.Run("locked after failures", func(t *testing.T) {
		q := newQuerierBuilder().
			onGetEmailVerificationCode(func(context.Context, sqlc.GetEmailVerificationCodeParams) (sqlc.EmailVerificationCode, error) {
				return sqlc.EmailVerificationCode{}, pgx.ErrNoRows
			}).
			build()

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

		body, _ := json.Marshal(verifyCodeRequest{Email: "user@example.com", Code: "ABCD2345"})
		req := httptest.NewRequest(http.MethodPost, "/auth/verify-code", bytes.NewReader(body))
		req.Header.Set("X-Device-Id", "device-123")
		rec := httptest.NewRecorder()

		api.handleVerifyCode(rec, req)

		if rec.Code != http.StatusTooManyRequests {
			t.Fatalf("expected status 429, got %d", rec.Code)
		}
	})

	t.Run("pre-locked", func(t *testing.T) {
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
	})

	t.Run("team full", func(t *testing.T) {
		email := "user@example.com"
		code := "ABCD2345"
		deviceID := "device-123"
		now := time.Now()

		q := newQuerierBuilder().
			onGetEmailVerificationCode(func(context.Context, sqlc.GetEmailVerificationCodeParams) (sqlc.EmailVerificationCode, error) {
				return sqlc.EmailVerificationCode{ID: pgtype.UUID{Bytes: [16]byte{1}, Valid: true}}, nil
			}).
			onMarkEmailVerificationCodeUsed(func(context.Context, sqlc.MarkEmailVerificationCodeUsedParams) error {
				return nil
			}).
			onGetUserByEmail(func(context.Context, string) (sqlc.User, error) {
				return sqlc.User{
					ID:              pgtype.UUID{Bytes: [16]byte{2}, Valid: true},
					Email:           email,
					EmailVerifiedAt: pgtype.Timestamptz{Time: now, Valid: true},
				}, nil
			}).
			onGetTeamByDomain(func(context.Context, string) (sqlc.Team, error) {
				return sqlc.Team{ID: pgtype.UUID{Bytes: [16]byte{3}, Valid: true}, Domain: "example.com", Name: "example.com"}, nil
			}).
			onGetTeamMembership(func(context.Context, sqlc.GetTeamMembershipParams) (sqlc.TeamMembership, error) {
				return sqlc.TeamMembership{}, pgx.ErrNoRows
			}).
			onCountTeamMembers(func(context.Context, pgtype.UUID) (int64, error) {
				return 1, nil
			}).
			build()

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
	})
}

func TestHandleRefresh(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		deviceID := "device-123"
		refreshToken := "refresh-token"
		now := time.Now()
		var rotated bool
		var created bool
		var marked bool

		q := newQuerierBuilder().
			onGetAuthSessionByRefreshHash(func(_ context.Context, _ sqlc.GetAuthSessionByRefreshHashParams) (sqlc.AuthSession, error) {
				return sqlc.AuthSession{
					ID:               pgtype.UUID{Bytes: [16]byte{1}, Valid: true},
					UserID:           pgtype.UUID{Bytes: [16]byte{2}, Valid: true},
					DeviceIDHash:     hashString(deviceID),
					RefreshTokenHash: hashString(refreshToken),
				}, nil
			}).
			onRotateAuthSession(func(context.Context, sqlc.RotateAuthSessionParams) error {
				rotated = true
				return nil
			}).
			onCreateAuthSession(func(context.Context, sqlc.CreateAuthSessionParams) (sqlc.AuthSession, error) {
				created = true
				return sqlc.AuthSession{}, nil
			}).
			onMarkAuthSessionUsed(func(context.Context, sqlc.MarkAuthSessionUsedParams) error {
				marked = true
				return nil
			}).
			build()

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
	})

	t.Run("within grace period does not rotate", func(t *testing.T) {
		deviceID := "device-123"
		refreshToken := "refresh-token"
		now := time.Now()
		var rotated bool

		q := newQuerierBuilder().
			onGetAuthSessionByRefreshHash(func(_ context.Context, _ sqlc.GetAuthSessionByRefreshHashParams) (sqlc.AuthSession, error) {
				return sqlc.AuthSession{
					ID:               pgtype.UUID{Bytes: [16]byte{1}, Valid: true},
					UserID:           pgtype.UUID{Bytes: [16]byte{2}, Valid: true},
					DeviceIDHash:     hashString(deviceID),
					RefreshTokenHash: hashString(refreshToken),
					RotatedAt:        pgtype.Timestamptz{Time: now.Add(-10 * time.Second), Valid: true},
				}, nil
			}).
			onRotateAuthSession(func(context.Context, sqlc.RotateAuthSessionParams) error {
				rotated = true
				return nil
			}).
			onCreateAuthSession(func(context.Context, sqlc.CreateAuthSessionParams) (sqlc.AuthSession, error) {
				return sqlc.AuthSession{}, nil
			}).
			onMarkAuthSessionUsed(func(context.Context, sqlc.MarkAuthSessionUsedParams) error {
				return nil
			}).
			build()

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
	})

	t.Run("invalid device", func(t *testing.T) {
		q := newQuerierBuilder().
			onGetAuthSessionByRefreshHash(func(_ context.Context, _ sqlc.GetAuthSessionByRefreshHashParams) (sqlc.AuthSession, error) {
				return sqlc.AuthSession{
					DeviceIDHash: hashString("other-device"),
				}, nil
			}).
			build()

		api := New(&stubStore{querier: q}, &mailer.LogMailer{}, Settings{
			AccessTTL:    15 * time.Minute,
			RefreshTTL:   24 * time.Hour,
			RefreshGrace: 30 * time.Second,
		}, nil)

		body, _ := json.Marshal(refreshRequest{RefreshToken: "refresh-token"})
		req := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader(body))
		req.Header.Set("X-Device-Id", "device-123")
		rec := httptest.NewRecorder()

		api.handleRefresh(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", rec.Code)
		}
	})

	t.Run("rotated token expired", func(t *testing.T) {
		deviceID := "device-123"
		refreshToken := "refresh-token"
		now := time.Now()

		q := newQuerierBuilder().
			onGetAuthSessionByRefreshHash(func(_ context.Context, _ sqlc.GetAuthSessionByRefreshHashParams) (sqlc.AuthSession, error) {
				return sqlc.AuthSession{
					DeviceIDHash:     hashString(deviceID),
					RefreshTokenHash: hashString(refreshToken),
					RotatedAt:        pgtype.Timestamptz{Time: now.Add(-time.Minute), Valid: true},
				}, nil
			}).
			build()

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
	})

	t.Run("not found", func(t *testing.T) {
		q := newQuerierBuilder().
			onGetAuthSessionByRefreshHash(func(context.Context, sqlc.GetAuthSessionByRefreshHashParams) (sqlc.AuthSession, error) {
				return sqlc.AuthSession{}, pgx.ErrNoRows
			}).
			build()

		api := New(&stubStore{querier: q}, &mailer.LogMailer{}, Settings{}, nil)

		body, _ := json.Marshal(refreshRequest{RefreshToken: "refresh-token"})
		req := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader(body))
		req.Header.Set("X-Device-Id", "device-123")
		rec := httptest.NewRecorder()

		api.handleRefresh(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", rec.Code)
		}
	})
}

func TestHandleRefreshInputValidation(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		deviceID string
	}{
		{"missing token", "", "device-123"},
		{"missing device", "refresh-token", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := New(&stubStore{}, &mailer.LogMailer{}, Settings{}, nil)
			body, _ := json.Marshal(refreshRequest{RefreshToken: tt.token})
			req := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader(body))
			if tt.deviceID != "" {
				req.Header.Set("X-Device-Id", tt.deviceID)
			}
			rec := httptest.NewRecorder()

			api.handleRefresh(rec, req)

			if rec.Code < 400 {
				t.Fatalf("expected error status, got %d", rec.Code)
			}
		})
	}
}

func TestHandleLogout(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		deviceID := "device-123"
		refreshToken := "refresh-token"
		var revoked bool

		q := newQuerierBuilder().
			onGetAuthSessionByRefreshHash(func(_ context.Context, _ sqlc.GetAuthSessionByRefreshHashParams) (sqlc.AuthSession, error) {
				return sqlc.AuthSession{
					ID:           pgtype.UUID{Bytes: [16]byte{1}, Valid: true},
					DeviceIDHash: hashString(deviceID),
				}, nil
			}).
			onRevokeAuthSession(func(context.Context, sqlc.RevokeAuthSessionParams) error {
				revoked = true
				return nil
			}).
			build()

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
	})

	t.Run("invalid device", func(t *testing.T) {
		q := newQuerierBuilder().
			onGetAuthSessionByRefreshHash(func(_ context.Context, _ sqlc.GetAuthSessionByRefreshHashParams) (sqlc.AuthSession, error) {
				return sqlc.AuthSession{
					DeviceIDHash: hashString("other-device"),
				}, nil
			}).
			build()

		api := New(&stubStore{querier: q}, &mailer.LogMailer{}, Settings{}, nil)

		body, _ := json.Marshal(refreshRequest{RefreshToken: "refresh-token"})
		req := httptest.NewRequest(http.MethodPost, "/auth/logout", bytes.NewReader(body))
		req.Header.Set("X-Device-Id", "device-123")
		rec := httptest.NewRecorder()

		api.handleLogout(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", rec.Code)
		}
	})
}

func TestHandleLogoutInputValidation(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		deviceID string
	}{
		{"missing token", "", "device-123"},
		{"missing device", "refresh-token", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := New(&stubStore{}, &mailer.LogMailer{}, Settings{}, nil)
			body, _ := json.Marshal(refreshRequest{RefreshToken: tt.token})
			req := httptest.NewRequest(http.MethodPost, "/auth/logout", bytes.NewReader(body))
			if tt.deviceID != "" {
				req.Header.Set("X-Device-Id", tt.deviceID)
			}
			rec := httptest.NewRecorder()

			api.handleLogout(rec, req)

			if rec.Code < 400 {
				t.Fatalf("expected error status, got %d", rec.Code)
			}
		})
	}
}

