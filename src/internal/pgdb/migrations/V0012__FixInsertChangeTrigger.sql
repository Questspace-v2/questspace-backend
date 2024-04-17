CREATE OR REPLACE FUNCTION questspace.insert_change() RETURNS TRIGGER AS $insert_change$
BEGIN
    IF (NEW.score > 0 AND NEW.accepted) THEN
        WITH quest_team AS (
            SELECT q.id AS quest_id
            FROM questspace.task task
                 LEFT JOIN questspace.task_group tg ON task.group_id = tg.id
                 LEFT JOIN questspace.quest q ON tg.quest_id = q.id
            WHERE task.id = NEW.task_id
        ) INSERT INTO questspace.score_change (quest_id, team_id, delta) VALUES ((SELECT quest_id FROM quest_team), NEW.team_id, NEW.score);
        RETURN NEW;
    END IF;
    RETURN NEW;
END;
$insert_change$ LANGUAGE plpgsql;
