CREATE TABLE questspace.permissions 
(
    user_id uuid NOT NULL REFERENCES questspace.user (id) ON DELETE CASCADE
);