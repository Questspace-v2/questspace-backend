CREATE TABLE questspace.team_penalty (
    id BIGSERIAL PRIMARY KEY,
    team_id uuid REFERENCES questspace.team (id) ON DELETE CASCADE,
    value integer
);
