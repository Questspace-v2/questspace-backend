CREATE TABLE questspace.hint_take (
    team_id uuid NOT NULL REFERENCES questspace.team (id) ON DELETE CASCADE,
    task_id uuid NOT NULL REFERENCES questspace.task (id) ON DELETE CASCADE,
    index integer CHECK ( index < 4 ),

    UNIQUE (task_id, team_id, index)
);

ALTER TABLE questspace.answer_try ADD COLUMN team_id uuid REFERENCES questspace.team (id) ON DELETE CASCADE;

ALTER TABLE questspace.answer_try ADD COLUMN try_time timestamp DEFAULT current_timestamp;

ALTER TABLE questspace.answer_try ADD COLUMN accepted bool DEFAULT false;

ALTER TABLE questspace.answer_try ADD COLUMN score integer;

ALTER TABLE questspace.answer_try DROP COLUMN user_id;

CREATE TABLE questspace.score_change (
    quest_id uuid NOT NULL REFERENCES questspace.quest (id) ON DELETE CASCADE,
    team_id uuid NOT NULL REFERENCES questspace.team (id) ON DELETE CASCADE,
    delta integer,
    time timestamp DEFAULT current_timestamp
);

CREATE FUNCTION questspace.insert_change() RETURNS TRIGGER AS $insert_change$
    BEGIN
        IF (NEW.score > 0 AND NEW.accepted) THEN
            WITH quest_team AS (SELECT q.id AS quest_id, t.id AS team_id
                                FROM questspace.task task
                                    LEFT JOIN questspace.task_group tg ON task.group_id= tg.id
                                    LEFT JOIN questspace.quest q ON tg.quest_id = q.id
                                    LEFT JOIN questspace.registration r ON r.user_id = NEW.user_id
                                    LEFT JOIN questspace.team t ON t.id = r.team_id
                                WHERE task.id = NEW.task_id AND t.quest_id = q.id
            ) INSERT INTO questspace.score_change (quest_id, team_id, delta) VALUES (quest_team.quest_id, quest_team.team_id, NEW.score);
            RETURN NEW;
        END IF;
        RETURN NEW;
    END;
$insert_change$ LANGUAGE plpgsql;

CREATE TRIGGER audit_score_change
    AFTER INSERT OR UPDATE ON questspace.answer_try
    FOR EACH ROW EXECUTE FUNCTION questspace.insert_change();
