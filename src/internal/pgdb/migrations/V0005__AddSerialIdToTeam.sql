ALTER TABLE questspace.team DROP COLUMN invite_ink;
ALTER TABLE questspace.team ADD COLUMN invite_path varchar UNIQUE;

ALTER TABLE questspace.team ADD COLUMN link_id BIGSERIAL UNIQUE;
CREATE UNIQUE INDEX team_invite_path_idx ON questspace.team (invite_path);

ALTER TABLE questspace.team DROP COLUMN capitan;
ALTER TABLE questspace.team ADD COLUMN cap_id uuid REFERENCES questspace.user ON DELETE SET NULL;

ALTER TABLE questspace.team ADD CONSTRAINT unique_quest_id_name_constraint UNIQUE (quest_id, name);

CREATE FUNCTION register_cap() RETURNS TRIGGER AS $reg_cap_fn$
    BEGIN
        INSERT INTO questspace.registration (user_id, team_id) VALUES (NEW.cap_id, NEW.id);
        RETURN NEW;
    END; $reg_cap_fn$ LANGUAGE 'plpgsql';

CREATE TRIGGER register_cap_trigger
    AFTER INSERT
    ON questspace.team
    FOR EACH ROW
    EXECUTE FUNCTION register_cap();

CREATE FUNCTION check_under_capacity() RETURNS TRIGGER AS $check_und_cap$
    DECLARE
        member_count integer;
        max_cap integer;
    BEGIN
        SELECT INTO max_cap max_team_cap
        FROM questspace.quest q
            LEFT JOIN questspace.team t ON q.id = t.quest_id
        WHERE t.id = NEW.team_id;

        IF max_cap IS NULL
        THEN
            RETURN NEW;
        END IF;

        SELECT INTO member_count COUNT(*)
        FROM questspace.registration
        WHERE team_id = NEW.team_id;

        IF member_count = max_cap
        THEN
            RAISE EXCEPTION 'Max capacity reached';
        END IF;

        RETURN NEW;
    END; $check_und_cap$ LANGUAGE 'plpgsql';

CREATE TRIGGER check_team_under_cap_trigger
    BEFORE INSERT
    ON questspace.registration
    FOR EACH ROW
    EXECUTE FUNCTION check_under_capacity();
