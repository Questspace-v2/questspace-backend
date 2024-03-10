ALTER TABLE questspace.task_group ADD COLUMN order_idx integer NOT NULL DEFAULT 0;
ALTER TABLE questspace.task_group ADD CONSTRAINT unique_group_order UNIQUE (quest_id, order_idx) DEFERRABLE INITIALLY DEFERRED;

ALTER TABLE questspace.task ADD COLUMN order_idx integer NOT NULL DEFAULT 0;
ALTER TABLE questspace.task ADD CONSTRAINT unique_task_order UNIQUE (group_id, order_idx) DEFERRABLE INITIALLY DEFERRED;
