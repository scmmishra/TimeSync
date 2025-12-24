-- name: GetTeamByDomain :one
SELECT id, domain, name, created_at, updated_at
FROM teams
WHERE domain = $1;

-- name: CreateTeam :one
INSERT INTO teams (
    domain,
    name,
    created_at,
    updated_at
)
VALUES ($1, $2, now(), now())
RETURNING id, domain, name, created_at, updated_at;

-- name: CountTeamMembers :one
SELECT COUNT(*)
FROM team_memberships
WHERE team_id = $1;
