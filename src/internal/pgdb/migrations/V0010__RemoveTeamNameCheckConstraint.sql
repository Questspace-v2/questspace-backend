ALTER TABLE questspace.team DROP CONSTRAINT team_name_check;
ALTER TABLE questspace.team ADD CHECK ( length(name) > 0 );