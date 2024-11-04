ALTER TABLE questspace.task_group ADD COLUMN has_time_limit bool NOT NULL DEFAULT false;

ALTER TABLE questspace.task_group ADD COLUMN time_limit bigint DEFAULT NULL;
