ALTER TABLE questspace.quest ADD COLUMN creator_id uuid;

UPDATE questspace.quest q SET (creator_id) = ((
    SELECT u.id FROM questspace.user u WHERE u.username = q.creator
));

ALTER TABLE questspace.quest DROP CONSTRAINT quest_creator_fkey;
ALTER TABLE questspace.quest DROP COLUMN creator;
ALTER TABLE questspace.quest ADD CONSTRAINT quest_creator_id FOREIGN KEY (creator_id) REFERENCES questspace.user (id) ON DELETE SET NULL;
