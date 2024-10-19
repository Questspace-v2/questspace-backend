CREATE TYPE questspace.quest_type AS ENUM ('', 'ASSAULT', 'LINEAR');

ALTER TABLE questspace.quest ADD COLUMN quest_type questspace.quest_type NOT NULL DEFAULT '';

ALTER TABLE questspace.task_group ADD COLUMN sticky bool NOT NULL DEFAULT false;
