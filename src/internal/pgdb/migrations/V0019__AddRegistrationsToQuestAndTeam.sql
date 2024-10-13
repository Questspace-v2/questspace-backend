CREATE TYPE questspace.registration_type AS ENUM ('AUTO', 'VERIFY');

ALTER TABLE questspace.quest ADD COLUMN registration_type questspace.registration_type NOT NULL DEFAULT 'AUTO';

CREATE TYPE questspace.registration_status AS ENUM ('ON_CONSIDERATION', 'ACCEPTED');

ALTER TABLE questspace.team ADD COLUMN registration_status questspace.registration_status NOT NULL DEFAULT 'ACCEPTED';
