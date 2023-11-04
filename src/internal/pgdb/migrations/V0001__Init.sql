CREATE SCHEMA questspace;

CREATE TYPE questspace.access_type AS ENUM ('public', 'link_only');

CREATE TYPE questspace.verification_type AS ENUM ('auto', 'manual');

CREATE TABLE questspace.quest
(
    id uuid DEFAULT gen_random_uuid() PRIMARY KEY,
    name varchar(40) NOT NULL CHECK ( length(name) > 0 ),
    description varchar,
    access questspace.access_type DEFAULT 'link_only',
    creator varchar NOT NULL,
    registration_deadline timestamp,
    start_time timestamp NOT NULL,
    finish_time timestamp,
    media_link varchar,
    max_team_cap integer CHECK ( max_team_cap > 0 )
);

CREATE TABLE questspace.team
(
    id uuid DEFAULT gen_random_uuid() PRIMARY KEY,
    name varchar(26) NOT NULL CHECK ( length(name) > 2 ),
    quest_id uuid NOT NULL,
    capitan varchar,
    score integer DEFAULT 0,
    invite_ink varchar NOT NULL,

    FOREIGN KEY (quest_id) REFERENCES questspace.quest (id) ON DELETE CASCADE
);

CREATE TABLE questspace.user
(
    id uuid DEFAULT gen_random_uuid() PRIMARY KEY,
    username varchar UNIQUE NOT NULL CHECK ( length(username) > 0 ),
    password varchar NOT NULL,
    first_name varchar,
    last_name varchar,
    avatar_url varchar
);

CREATE UNIQUE INDEX ON questspace.user (username);

CREATE TABLE questspace.registration
(
    user_id uuid NOT NULL,
    team_id uuid NOT NULL,
    PRIMARY KEY (user_id, team_id),

    FOREIGN KEY (user_id) REFERENCES questspace.user (id) ON DELETE CASCADE,
    FOREIGN KEY (team_id) REFERENCES questspace.team (id) ON DELETE CASCADE
);

CREATE TABLE questspace.task_group
(
    id uuid DEFAULT gen_random_uuid() PRIMARY KEY,
    quest_id uuid NOT NULL,
    name varchar NOT NULL,
    pub_time timestamp,

    FOREIGN KEY (quest_id) REFERENCES questspace.quest (id) ON DELETE CASCADE
);

CREATE TABLE questspace.task
(
    id uuid DEFAULT gen_random_uuid() PRIMARY KEY,
    group_id uuid NOT NULL,
    name varchar NOT NULL,
    question varchar NOT NULL,
    reward integer NOT NULL,
    correct_answers varchar[],
    verification questspace.verification_type DEFAULT 'auto',
    hints varchar[] CHECK ( cardinality(hints) < 4 ),
    pub_time timestamp,
    media_url varchar,

    FOREIGN KEY (group_id) REFERENCES questspace.task_group (id) ON DELETE CASCADE
);

CREATE TABLE questspace.answer_try
(
    task_id uuid NOT NULL,
    user_id uuid NOT NULL,
    answer varchar NOT NULL,

    FOREIGN KEY (task_id) REFERENCES questspace.task (id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES questspace.user (id) ON DELETE CASCADE
);

ALTER TABLE questspace.quest ADD FOREIGN KEY (creator) REFERENCES questspace.user (username);

ALTER TABLE questspace.team ADD FOREIGN KEY (capitan) REFERENCES questspace.user (username);
