ALTER TABLE questspace.answer_try ADD COLUMN user_id uuid REFERENCES questspace.user (id);

CREATE INDEX answer_try_try_time_idx ON questspace.answer_try (try_time);
