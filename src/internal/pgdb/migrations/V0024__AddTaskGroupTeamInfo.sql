CREATE TABLE questspace.task_group_team_info (
    team_id uuid NOT NULL REFERENCES questspace.team (id) ON DELETE CASCADE,
    group_id uuid NOT NULL REFERENCES questspace.task_group (id) ON DELETE CASCADE,
    opening_time timestamp NOT NULL DEFAULT to_timestamp(0),
    closing_time timestamp DEFAULT NULL,

    UNIQUE (team_id, group_id)
);
