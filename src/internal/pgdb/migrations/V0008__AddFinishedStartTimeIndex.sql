CREATE INDEX finished_start_time_quest_idx ON questspace.quest USING btree (finished, start_time);
