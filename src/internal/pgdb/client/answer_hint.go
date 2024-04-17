package pgdb

import (
	"context"
	"database/sql"
	"errors"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
	"golang.org/x/xerrors"

	"questspace/pkg/application/logging"
	"questspace/pkg/storage"
)

//func (c *Client) CreateAnswerTry(ctx context.Context, req *storage.CreateAnswerTryRequest) error {
//	query := sq.Insert("questspace.answer_try").
//		Columns("task_id", "user_id", "answer").
//		Values(req.TaskID, req.UserID, req.Text).
//		PlaceholderFormat(sq.Dollar)
//
//	if _, err := query.RunWith(c.runner).ExecContext(ctx); err != nil {
//		return xerrors.Errorf("executing query: %w", err)
//	}
//	return nil
//}
//
//func (c *Client) GetAnswerTries(ctx context.Context, req *storage.GetAnswerTriesRequest) ([]storage.AnswerTry, error) {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (c *Client) TakeHint(ctx context.Context, req *storage.TakeHintRequest) (string, error) {
//	//if req.TeamID != "" {
//	//	query := sq.Insert("questspace.answer_try").
//	//		Columns("team_id", "task_id", "answer").
//	//		Values(req.TeamID, req.TaskID, req.)
//	//}
//	//	sqlQuery := sq.Expr(`INSERT INTO questspace.hint_take (team_id, task_id, index) VALUES ((
//	//	SELECT t.id FROM questspace.registration r LEFT JOIN questspace.team t on t.id = r.team_id
//	//	WHERE r.user_id = $1 AND t.quest_id = $2
//	//) AS got_team_id, $3, $4)`, req.UserID, req.QuestID, req.TaskID, req.Index)
//	//
//	//	if _, err := sq.ExecContextWith(ctx, c.runner, sqlQuery); err != nil {
//	//		if pgErr := new(pgconn.PgError); errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
//	//			return storage.ErrExists
//	//		}
//	//		return xerrors.Errorf("exec query: %w", err)
//	//	}
//	//	return nil
//	panic("implement me")
//}
//
//func (c *Client) GetHintTakes(ctx context.Context, req *storage.GetHintTakesRequest) ([]storage.Hint, string, error) {
//
//	//TODO implement me
//	panic("implement me")
//}

func (c *Client) GetHintTakes(ctx context.Context, req *storage.GetHintTakesRequest) (storage.HintTakes, error) {
	query := sq.Select("ht.task_id", "ht.index", "t.hints").
		From("questspace.hint_take ht").LeftJoin("questspace.task t ON ht.task_id = t.id").
		LeftJoin("questspace.task_group tg ON t.group_id = tg.id").
		Where(sq.Eq{"tg.quest_id": req.QuestID, "ht.team_id": req.TeamID}).
		PlaceholderFormat(sq.Dollar)
	if req.TaskID != "" {
		query = query.Where(sq.Eq{"ht.task_id": req.TaskID})
	}

	rows, err := query.RunWith(c.runner).QueryContext(ctx)
	if err != nil {
		return nil, xerrors.Errorf("query rows: %w", err)
	}
	defer func() { _ = rows.Close() }()

	pgMap := pgtype.NewMap()
	hintTakes := make(storage.HintTakes)
	for rows.Next() {
		var ht storage.HintTake
		var allHints []string
		if err = rows.Scan(&ht.TaskID, &ht.Hint.Index, pgMap.SQLScanner(&allHints)); err != nil {
			return nil, xerrors.Errorf("scan row: %w", err)
		}
		if len(allHints) < ht.Hint.Index {
			logging.Error(ctx, "took hint with index more than amount of hints", zap.String("task_id", ht.TaskID))
			continue
		}
		ht.Hint.Text = allHints[ht.Hint.Index]
		tookHints := hintTakes[ht.TaskID]
		tookHints = append(tookHints, ht)
		hintTakes[ht.TaskID] = tookHints
	}
	if err := rows.Err(); err != nil {
		return nil, xerrors.Errorf("iter rows: %w", err)
	}

	return hintTakes, nil
}

func (c *Client) TakeHint(ctx context.Context, req *storage.TakeHintRequest) (*storage.Hint, error) {
	sqlQuery := `
WITH inserted_hint AS (
    INSERT INTO questspace.hint_take (team_id, task_id, index) VALUES ($1, $2, $3)
        ON CONFLICT (task_id, team_id, index) DO NOTHING
        RETURNING task_id, index
) SELECT t.hints, inserted_hint.index FROM inserted_hint
    LEFT JOIN questspace.task t ON inserted_hint.task_id = t.id
`
	query := sq.Expr(sqlQuery, req.TeamID, req.TaskID, req.Index)
	row := sq.QueryRowContextWith(ctx, c.runner, query)

	pgMap := pgtype.NewMap()
	var h storage.Hint
	var hints []string
	if err := row.Scan(pgMap.SQLScanner(&hints), &h.Index); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, xerrors.Errorf("scan row: %w", err)
	}
	h.Text = hints[h.Index]
	return &h, nil
}

func (c *Client) GetAcceptedTasks(ctx context.Context, req *storage.GetAcceptedTasksRequest) (storage.AcceptedTasks, error) {
	query := sq.Select("t.id").
		From("questspace.answer_try at").
		LeftJoin("questspace.task t ON at.task_id = t.id").
		LeftJoin("questspace.task_group tg ON t.group_id = tg.id").
		Where(sq.Eq{"at.team_id": req.TeamID, "tg.quest_id": req.QuestID, "at.accepted": true}).
		PlaceholderFormat(sq.Dollar)

	rows, err := query.RunWith(c.runner).QueryContext(ctx)
	if err != nil {
		return nil, xerrors.Errorf("query rows: %w", err)
	}
	defer func() { _ = rows.Close() }()

	acceptedTasks := make(storage.AcceptedTasks)
	for rows.Next() {
		id := ""
		if err = rows.Scan(&id); err != nil {
			return nil, xerrors.Errorf("scan row: %w", err)
		}
		acceptedTasks[id] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, xerrors.Errorf("iter rows: %w", err)
	}

	return acceptedTasks, nil
}

func (c *Client) CreateAnswerTry(ctx context.Context, req *storage.CreateAnswerTryRequest) error {
	query := sq.Insert("questspace.answer_try").
		Columns("team_id", "task_id", "answer", "accepted").
		Values(req.TeamID, req.TaskID, req.Text, req.Accepted).
		PlaceholderFormat(sq.Dollar)
	if req.Score > 0 {
		query = query.Columns("score").Values(req.Score)
	}

	if _, err := query.RunWith(c.runner).ExecContext(ctx); err != nil {
		return xerrors.Errorf("exec query: %w", err)
	}
	return nil
}
