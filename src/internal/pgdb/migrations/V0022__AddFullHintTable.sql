CREATE TABLE questspace.hint (
    task_id uuid NOT NULL REFERENCES questspace.task (id) ON DELETE CASCADE,
    index integer CHECK ( index < 4 ),
    name varchar,
    text varchar NOT NULL,
    penalty_score integer,
    penalty_percent integer,

    UNIQUE (index, task_id)
);

INSERT INTO questspace.hint (task_id, index, text, penalty_percent) SELECT 
    t.id as task_id, 
    hint_ord.nr::integer-1 as index,
    hint_ord.elem as text,
    20 as penalty_percent
FROM questspace.task t LEFT JOIN LATERAL UNNEST (t.hints) WITH ORDINALITY AS hint_ord(elem, nr) ON true
    WHERE hint_ord IS NOT NULL;

ALTER TYPE questspace.quest_type ADD VALUE 'LINEAR_TOTAL';
