CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email text NOT NULL UNIQUE,
    email_domain text NOT NULL,
    email_verified_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX users_email_domain_idx ON users (email_domain);

CREATE TABLE teams (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    domain text NOT NULL UNIQUE,
    name text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE team_memberships (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id uuid NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role text NOT NULL CHECK (role IN ('admin', 'member')),
    joined_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (team_id, user_id)
);

CREATE INDEX team_memberships_team_id_idx ON team_memberships (team_id);
CREATE INDEX team_memberships_user_id_idx ON team_memberships (user_id);

CREATE TABLE invite_codes (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id uuid NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    email text NOT NULL,
    code text NOT NULL UNIQUE,
    expires_at timestamptz NOT NULL,
    redeemed_at timestamptz NULL,
    created_by_user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX invite_codes_team_id_idx ON invite_codes (team_id);
CREATE INDEX invite_codes_email_idx ON invite_codes (email);

CREATE TABLE timezone_states (
    user_id uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    timezone text NOT NULL,
    utc_offset_minutes integer NOT NULL,
    country_code text NOT NULL,
    reported_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX timezone_states_reported_at_idx ON timezone_states (reported_at);

CREATE TABLE working_hours (
    user_id uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    start_minute integer NOT NULL,
    end_minute integer NOT NULL,
    saturday_enabled boolean NOT NULL,
    sunday_enabled boolean NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE timezone_visibility (
    user_id uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    hidden_until timestamptz NULL,
    hidden_indefinitely boolean NOT NULL DEFAULT false,
    last_reminder_at timestamptz NULL,
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX timezone_visibility_hidden_until_idx ON timezone_visibility (hidden_until);

CREATE TABLE email_verification_codes (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email text NOT NULL,
    code text NOT NULL,
    expires_at timestamptz NOT NULL,
    used_at timestamptz NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (email, code)
);

CREATE INDEX email_verification_codes_email_idx ON email_verification_codes (email);
