-- name: CreateAuthSession :one
INSERT INTO auth_sessions (
    user_id,
    device_id_hash,
    access_token_hash,
    access_expires_at,
    refresh_token_hash,
    refresh_expires_at,
    created_at
)
VALUES ($1, $2, $3, $4, $5, $6, now())
RETURNING id, user_id, device_id_hash, access_token_hash, access_expires_at,
          refresh_token_hash, refresh_expires_at, rotated_at, revoked_at,
          last_used_at, created_at;

-- name: GetAuthSessionByAccessHash :one
SELECT id, user_id, device_id_hash, access_token_hash, access_expires_at,
       refresh_token_hash, refresh_expires_at, rotated_at, revoked_at,
       last_used_at, created_at
FROM auth_sessions
WHERE access_token_hash = $1
  AND access_expires_at > $2
  AND revoked_at IS NULL;

-- name: GetAuthSessionByRefreshHash :one
SELECT id, user_id, device_id_hash, access_token_hash, access_expires_at,
       refresh_token_hash, refresh_expires_at, rotated_at, revoked_at,
       last_used_at, created_at
FROM auth_sessions
WHERE refresh_token_hash = $1
  AND refresh_expires_at > $2
  AND revoked_at IS NULL;

-- name: MarkAuthSessionUsed :exec
UPDATE auth_sessions
SET last_used_at = $2
WHERE id = $1;

-- name: RotateAuthSession :exec
UPDATE auth_sessions
SET rotated_at = $2
WHERE id = $1
  AND rotated_at IS NULL
  AND revoked_at IS NULL;

-- name: RevokeAuthSession :exec
UPDATE auth_sessions
SET revoked_at = $2
WHERE id = $1;
