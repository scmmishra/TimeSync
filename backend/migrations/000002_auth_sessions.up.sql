CREATE TABLE auth_sessions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_id_hash bytea NOT NULL,
    access_token_hash bytea NOT NULL,
    access_expires_at timestamptz NOT NULL,
    refresh_token_hash bytea NOT NULL,
    refresh_expires_at timestamptz NOT NULL,
    rotated_at timestamptz NULL,
    revoked_at timestamptz NULL,
    last_used_at timestamptz NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX auth_sessions_user_id_idx ON auth_sessions (user_id);
CREATE INDEX auth_sessions_access_token_hash_idx ON auth_sessions (access_token_hash);
CREATE INDEX auth_sessions_refresh_token_hash_idx ON auth_sessions (refresh_token_hash);
