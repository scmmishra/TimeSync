-- name: CreateTeamMembership :exec
INSERT INTO team_memberships (
    team_id,
    user_id,
    role,
    joined_at,
    created_at
)
VALUES ($1, $2, $3, $4, now())
ON CONFLICT (team_id, user_id) DO NOTHING;

-- name: GetTeamMembership :one
SELECT id, team_id, user_id, role, joined_at, created_at
FROM team_memberships
WHERE team_id = $1
  AND user_id = $2;
