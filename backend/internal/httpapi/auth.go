package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"timesync/backend/internal/sqlc"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type requestCodeRequest struct {
	Email string `json:"email"`
}

type verifyCodeRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type authResponse struct {
	AccessToken      string         `json:"access_token"`
	AccessExpiresAt  time.Time      `json:"access_expires_at"`
	RefreshToken     string         `json:"refresh_token"`
	RefreshExpiresAt time.Time      `json:"refresh_expires_at"`
	User             *userResponse  `json:"user,omitempty"`
	Team             *teamResponse  `json:"team,omitempty"`
	Role             string         `json:"role,omitempty"`
}

type userResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

type teamResponse struct {
	ID     string `json:"id"`
	Domain string `json:"domain"`
	Name   string `json:"name"`
}

func (a *API) handleRequestCode(w http.ResponseWriter, r *http.Request) {
	var req requestCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	email := normalizeEmail(req.Email)
	if email == "" {
		writeError(w, http.StatusBadRequest, "email is required")
		return
	}

	now := a.clock()
	if !a.emailLimit.Allow(email, requestCodeLimit, 15*time.Minute, now) {
		writeError(w, http.StatusTooManyRequests, "too many requests")
		return
	}

	code, err := generateCode()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate code")
		return
	}

	expiresAt := now.Add(codeTTL)
	_, err = a.store.Queries.CreateEmailVerificationCode(r.Context(), sqlc.CreateEmailVerificationCodeParams{
		Email:     email,
		Code:      code,
		ExpiresAt: toTimestamptz(expiresAt),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create verification code")
		return
	}

	if err := a.mailer.SendVerificationCode(r.Context(), email, code); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to send verification code")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (a *API) handleVerifyCode(w http.ResponseWriter, r *http.Request) {
	var req verifyCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	email := normalizeEmail(req.Email)
	if email == "" || strings.TrimSpace(req.Code) == "" {
		writeError(w, http.StatusBadRequest, "email and code are required")
		return
	}

	deviceID := strings.TrimSpace(r.Header.Get("X-Device-Id"))
	if deviceID == "" {
		writeError(w, http.StatusBadRequest, "X-Device-Id is required")
		return
	}

	now := a.clock()
	if a.failLimit.IsLocked(email, now) {
		writeError(w, http.StatusTooManyRequests, "too many attempts")
		return
	}

	ctx := r.Context()
	tx, err := a.store.Pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to start transaction")
		return
	}
	defer tx.Rollback(ctx)

	q := a.store.Queries.WithTx(tx)
	codeRow, err := q.GetEmailVerificationCode(ctx, sqlc.GetEmailVerificationCodeParams{
		Email:     email,
		Code:      strings.TrimSpace(req.Code),
		ExpiresAt: toTimestamptz(now),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			locked := a.failLimit.RegisterFailure(email, verifyCodeLimit, 15*time.Minute, 15*time.Minute, now)
			if locked {
				writeError(w, http.StatusTooManyRequests, "too many attempts")
				return
			}
			writeError(w, http.StatusUnauthorized, "invalid code")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to verify code")
		return
	}

	if err := q.MarkEmailVerificationCodeUsed(ctx, sqlc.MarkEmailVerificationCodeUsedParams{
		ID:     codeRow.ID,
		UsedAt: toTimestamptz(now),
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to verify code")
		return
	}

	domain, ok := emailDomain(email)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid email")
		return
	}

	user, isNewUser, err := getOrCreateUser(ctx, q, email, domain, now)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	team, createdTeam, err := getOrCreateTeam(ctx, q, domain)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve team")
		return
	}

	role, err := ensureMembership(ctx, q, user, team, isNewUser, createdTeam, now)
	if err != nil {
		if errors.Is(err, errTeamFull) {
			writeError(w, http.StatusConflict, "team is full")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create membership")
		return
	}

	accessToken, accessHash, err := generateToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to issue token")
		return
	}
	refreshToken, refreshHash, err := generateToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to issue token")
		return
	}

	accessExpires := now.Add(accessTTL)
	refreshExpires := now.Add(refreshTTL)
	_, err = q.CreateAuthSession(ctx, sqlc.CreateAuthSessionParams{
		UserID:           user.ID,
		DeviceIDHash:     hashString(deviceID),
		AccessTokenHash:  accessHash,
		AccessExpiresAt:  toTimestamptz(accessExpires),
		RefreshTokenHash: refreshHash,
		RefreshExpiresAt: toTimestamptz(refreshExpires),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to issue token")
		return
	}

	if err := tx.Commit(ctx); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save session")
		return
	}

	a.failLimit.Reset(email)

	writeJSON(w, http.StatusOK, authResponse{
		AccessToken:      accessToken,
		AccessExpiresAt:  accessExpires,
		RefreshToken:     refreshToken,
		RefreshExpiresAt: refreshExpires,
		User: &userResponse{
			ID:    uuidString(user.ID),
			Email: user.Email,
		},
		Team: &teamResponse{
			ID:     uuidString(team.ID),
			Domain: team.Domain,
			Name:   team.Name,
		},
		Role: role,
	})
}

func (a *API) handleRefresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	refreshToken := strings.TrimSpace(req.RefreshToken)
	if refreshToken == "" {
		writeError(w, http.StatusBadRequest, "refresh_token is required")
		return
	}

	deviceID := strings.TrimSpace(r.Header.Get("X-Device-Id"))
	if deviceID == "" {
		writeError(w, http.StatusBadRequest, "X-Device-Id is required")
		return
	}

	now := a.clock()
	session, err := a.store.Queries.GetAuthSessionByRefreshHash(r.Context(), sqlc.GetAuthSessionByRefreshHashParams{
		RefreshTokenHash: hashString(refreshToken),
		RefreshExpiresAt: toTimestamptz(now),
	})
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}

	if !hashEqual(session.DeviceIDHash, hashString(deviceID)) {
		writeError(w, http.StatusUnauthorized, "invalid device")
		return
	}

	if session.RotatedAt.Valid {
		if now.Sub(session.RotatedAt.Time) > refreshGrace {
			writeError(w, http.StatusUnauthorized, "refresh token expired")
			return
		}
	} else {
		_ = a.store.Queries.RotateAuthSession(r.Context(), sqlc.RotateAuthSessionParams{
			ID:        session.ID,
			RotatedAt: toTimestamptz(now),
		})
	}

	accessToken, accessHash, err := generateToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to issue token")
		return
	}
	newRefreshToken, refreshHash, err := generateToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to issue token")
		return
	}

	accessExpires := now.Add(accessTTL)
	refreshExpires := now.Add(refreshTTL)
	_, err = a.store.Queries.CreateAuthSession(r.Context(), sqlc.CreateAuthSessionParams{
		UserID:           session.UserID,
		DeviceIDHash:     session.DeviceIDHash,
		AccessTokenHash:  accessHash,
		AccessExpiresAt:  toTimestamptz(accessExpires),
		RefreshTokenHash: refreshHash,
		RefreshExpiresAt: toTimestamptz(refreshExpires),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to issue token")
		return
	}

	_ = a.store.Queries.MarkAuthSessionUsed(r.Context(), sqlc.MarkAuthSessionUsedParams{
		ID:         session.ID,
		LastUsedAt: toTimestamptz(now),
	})

	writeJSON(w, http.StatusOK, authResponse{
		AccessToken:      accessToken,
		AccessExpiresAt:  accessExpires,
		RefreshToken:     newRefreshToken,
		RefreshExpiresAt: refreshExpires,
	})
}

func (a *API) handleLogout(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	refreshToken := strings.TrimSpace(req.RefreshToken)
	if refreshToken == "" {
		writeError(w, http.StatusBadRequest, "refresh_token is required")
		return
	}

	deviceID := strings.TrimSpace(r.Header.Get("X-Device-Id"))
	if deviceID == "" {
		writeError(w, http.StatusBadRequest, "X-Device-Id is required")
		return
	}

	now := a.clock()
	session, err := a.store.Queries.GetAuthSessionByRefreshHash(r.Context(), sqlc.GetAuthSessionByRefreshHashParams{
		RefreshTokenHash: hashString(refreshToken),
		RefreshExpiresAt: toTimestamptz(now),
	})
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}

	if !hashEqual(session.DeviceIDHash, hashString(deviceID)) {
		writeError(w, http.StatusUnauthorized, "invalid device")
		return
	}

	_ = a.store.Queries.RevokeAuthSession(r.Context(), sqlc.RevokeAuthSessionParams{
		ID:        session.ID,
		RevokedAt: toTimestamptz(now),
	})

	w.WriteHeader(http.StatusNoContent)
}

func normalizeEmail(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func emailDomain(email string) (string, bool) {
	at := strings.LastIndex(email, "@")
	if at <= 0 || at == len(email)-1 {
		return "", false
	}
	return email[at+1:], true
}

func toTimestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

func uuidString(id pgtype.UUID) string {
	if !id.Valid {
		return ""
	}
	return id.String()
}

func hashEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var out byte
	for i := 0; i < len(a); i++ {
		out |= a[i] ^ b[i]
	}
	return out == 0
}

var errTeamFull = errors.New("team is full")

func getOrCreateUser(ctx context.Context, q *sqlc.Queries, email, domain string, now time.Time) (sqlc.User, bool, error) {
	user, err := q.GetUserByEmail(ctx, email)
	if err == nil {
		if !user.EmailVerifiedAt.Valid {
			user, err = q.UpdateUserVerifiedAt(ctx, sqlc.UpdateUserVerifiedAtParams{
				ID:              user.ID,
				EmailVerifiedAt: toTimestamptz(now),
			})
			if err != nil {
				return sqlc.User{}, false, err
			}
		}
		return user, false, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return sqlc.User{}, false, err
	}
	user, err = q.CreateUser(ctx, sqlc.CreateUserParams{
		Email:           email,
		EmailDomain:     domain,
		EmailVerifiedAt: toTimestamptz(now),
	})
	if err != nil {
		return sqlc.User{}, false, err
	}
	return user, true, nil
}

func getOrCreateTeam(ctx context.Context, q *sqlc.Queries, domain string) (sqlc.Team, bool, error) {
	team, err := q.GetTeamByDomain(ctx, domain)
	if err == nil {
		return team, false, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return sqlc.Team{}, false, err
	}
	team, err = q.CreateTeam(ctx, sqlc.CreateTeamParams{
		Domain: domain,
		Name:   domain,
	})
	if err != nil {
		return sqlc.Team{}, false, err
	}
	return team, true, nil
}

func ensureMembership(ctx context.Context, q *sqlc.Queries, user sqlc.User, team sqlc.Team, isNewUser, createdTeam bool, now time.Time) (string, error) {
	membership, err := q.GetTeamMembership(ctx, sqlc.GetTeamMembershipParams{
		TeamID: team.ID,
		UserID: user.ID,
	})
	if err == nil {
		return membership.Role, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}

	count, err := q.CountTeamMembers(ctx, team.ID)
	if err != nil {
		return "", err
	}
	if count >= 30 {
		return "", errTeamFull
	}

	role := "member"
	if createdTeam || isNewUser && count == 0 {
		role = "admin"
	}

	if err := q.CreateTeamMembership(ctx, sqlc.CreateTeamMembershipParams{
		TeamID:   team.ID,
		UserID:   user.ID,
		Role:     role,
		JoinedAt: toTimestamptz(now),
	}); err != nil {
		return "", err
	}

	return role, nil
}
