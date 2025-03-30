package pgclient

import (
	"context"
	"database/sql"
	"errors"

	sq "github.com/Masterminds/squirrel"
	"github.com/yandex/perforator/library/go/core/xerrors"

	"questspace/internal/qtime"
	"questspace/pkg/storage"
)

func (c *Client) GetHintTakes(ctx context.Context, req *storage.GetHintTakesRequest) (storage.HintTakes, error) {
	query := sq.Select(
		"ht.task_id",
		"ht.index",
		"h.name",
		"h.text",
		"h.penalty_score",
		"h.penalty_percent",
	).
		From("questspace.hint_take ht").
		LeftJoin("questspace.task t ON ht.task_id = t.id").
		LeftJoin("questspace.task_group tg ON t.group_id = tg.id").
		LeftJoin("questspace.hint h ON ht.task_id = h.task_id AND ht.index = h.index").
		Where(sq.Eq{
			"tg.quest_id": req.QuestID,
			"ht.team_id":  req.TeamID,
		}).
		OrderBy("ht.index").
		PlaceholderFormat(sq.Dollar)
	if req.TaskID != "" {
		query = query.Where(sq.Eq{"ht.task_id": req.TaskID})
	}

	rows, err := query.RunWith(c.runner).QueryContext(ctx)
	if err != nil {
		return nil, xerrors.Errorf("query rows: %w", err)
	}
	defer func() { _ = rows.Close() }()

	hintsByTaskID := make(storage.HintTakes)
	for rows.Next() {
		var ht storage.HintTake
		var percent, score *int

		if err = rows.Scan(
			&ht.TaskID,
			&ht.Hint.Index,
			&ht.Hint.Name,
			&ht.Hint.Text,
			&score,
			&percent,
		); err != nil {
			return nil, xerrors.Errorf("scan row: %w", err)
		}

		if percent != nil {
			ht.Hint.Penalty, err = storage.NewPercentagePenalty(*percent)
			if err != nil {
				return nil, xerrors.Errorf("bad penalty: %w", err)
			}
		}
		if score != nil {
			ht.Hint.Penalty = storage.NewScorePenalty(*score)
		}

		takenHints := hintsByTaskID[ht.TaskID]
		takenHints = append(takenHints, ht)
		hintsByTaskID[ht.TaskID] = takenHints
	}
	if err := rows.Err(); err != nil {
		return nil, xerrors.Errorf("iter rows: %w", err)
	}

	return hintsByTaskID, nil
}

func (c *Client) TakeHint(ctx context.Context, req *storage.TakeHintRequest) (*storage.Hint, error) {
	sqlQuery := `
WITH inserted_hint AS (
    INSERT INTO questspace.hint_take (team_id, task_id, index) VALUES ($1, $2, $3)
        ON CONFLICT (task_id, team_id, index) DO UPDATE SET index = $3
        RETURNING task_id, index
) SELECT inserted_hint.index, h.name, h.text, h.penalty_score, h.penalty_percent FROM inserted_hint
    LEFT JOIN questspace.hint h ON inserted_hint.task_id = h.task_id AND inserted_hint.index = h.index
`
	query := sq.Expr(sqlQuery, req.TeamID, req.TaskID, req.Index)
	row := sq.QueryRowContextWith(ctx, c.runner, query)

	var h storage.Hint
	var score, percent *int
	if err := row.Scan(
		&h.Index,
		&h.Name,
		&h.Text,
		&score,
		&percent,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, xerrors.Errorf("scan row: %w", err)
	}

	var err error
	if percent != nil {
		h.Penalty, err = storage.NewPercentagePenalty(*percent)
		if err != nil {
			return nil, xerrors.Errorf("bad penalty: %w", err)
		}
	}
	if score != nil {
		h.Penalty = storage.NewScorePenalty(*score)
	}

	return &h, nil
}

func (c *Client) GetAcceptedTasks(ctx context.Context, req *storage.GetAcceptedTasksRequest) (storage.AcceptedTasks, error) {
	query := sq.Select("t.id", "at.answer", "at.score").
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
		var task storage.AcceptedTask
		var id storage.ID
		if err = rows.Scan(&id, &task.Text, &task.Score); err != nil {
			return nil, xerrors.Errorf("scan row: %w", err)
		}
		acceptedTasks[id] = task
	}
	if err = rows.Err(); err != nil {
		return nil, xerrors.Errorf("iter rows: %w", err)
	}

	return acceptedTasks, nil
}

func (c *Client) CreateAnswerTry(ctx context.Context, req *storage.CreateAnswerTryRequest) error {
	query := `
	INSERT INTO questspace.answer_try (team_id, user_id, task_id, answer, accepted, score, try_time)
	VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	if _, err := c.runner.ExecContext(
		ctx, query,
		req.TeamID,
		req.UserID,
		req.TaskID,
		req.Text,
		req.Accepted,
		req.Score,
		qtime.Now(),
	); err != nil {
		return xerrors.Errorf("exec query: %w", err)
	}
	return nil
}

func (c *Client) GetScoreResults(ctx context.Context, req *storage.GetResultsRequest) (storage.ScoreResults, error) {
	query := sq.Select("tm.id", "tm.name", "tg.id", "tg.name", "t.id", "t.name", "at.score", "at.try_time").
		From("questspace.team tm").
		LeftJoin("questspace.answer_try at ON at.team_id = tm.id").
		LeftJoin("questspace.task t ON at.task_id = t.id").
		LeftJoin("questspace.task_group tg ON t.group_id = tg.id").
		Where(sq.Eq{"at.accepted": true}).
		PlaceholderFormat(sq.Dollar)
	if req.QuestID != "" {
		query = query.Where(sq.Eq{"tg.quest_id": req.QuestID})
	}
	if len(req.TeamIDs) > 0 {
		query = query.Where(sq.Eq{"tg.team_id": req.TeamIDs})
	}

	rows, err := query.RunWith(c.runner).QueryContext(ctx)
	if err != nil {
		return nil, xerrors.Errorf("query rows: %w", err)
	}
	defer func() { _ = rows.Close() }()

	scoreRes := make(storage.ScoreResults)
	for rows.Next() {
		var res storage.SingleTaskResult
		if err = rows.Scan(&res.TeamID, &res.TeamName, &res.GroupID, &res.GroupName, &res.TaskID, &res.TaskName, &res.Score, &res.ScoreTime); err != nil {
			return nil, xerrors.Errorf("scan row: %w", err)
		}
		taskRes := scoreRes[res.TeamID]
		if taskRes == nil {
			taskRes = make(map[storage.ID]storage.SingleTaskResult)
		}
		taskRes[res.TaskID] = res
		scoreRes[res.TeamID] = taskRes
	}
	if err := rows.Err(); err != nil {
		return nil, xerrors.Errorf("iter rows: %w", err)
	}

	return scoreRes, nil
}

const countOnly = "COUNT(*)"

func buildLogQuery(req *storage.GetAnswerTriesRequest, opts *storage.TaskRequestLogFilterOptions, fields ...string) sq.SelectBuilder {
	whereEq := sq.Eq{"tg.quest_id": req.QuestID}
	if len(opts.GroupID) > 0 {
		whereEq["tg.id"] = opts.GroupID
	}
	if len(opts.TaskID) > 0 {
		whereEq["t.id"] = opts.TaskID
	}
	if len(opts.TeamID) > 0 {
		whereEq["tm.id"] = opts.TeamID
	}
	if len(opts.UserID) > 0 {
		whereEq["u.id"] = opts.UserID
	}
	if opts.OnlyAccepted {
		whereEq["at.accepted"] = true
	}

	query := sq.Select(fields...).
		From("questspace.answer_try at").
		LeftJoin("questspace.team tm ON at.team_id = tm.id").
		LeftJoin("questspace.task t ON at.task_id = t.id").
		LeftJoin("questspace.task_group tg ON t.group_id = tg.id").
		LeftJoin("questspace.user u ON at.user_id = u.id").
		Where(whereEq).
		PlaceholderFormat(sq.Dollar)

	needsSorting := true
	if len(fields) == 1 && fields[0] == countOnly {
		needsSorting = false
	}

	if needsSorting && opts.DateDesc {
		query = query.OrderBy("at.try_time DESC")
	} else if needsSorting {
		query = query.OrderBy("at.try_time ASC")
	}

	return query
}

func (c *Client) GetAnswerTries(ctx context.Context, req *storage.GetAnswerTriesRequest, opts ...storage.FilteringOption) (*storage.AnswerLogRecords, error) {
	options := storage.NewDefaultLogOpts()
	for _, opt := range opts {
		opt(&options)
	}

	countQuery := buildLogQuery(req, &options, countOnly)
	countRow := countQuery.RunWith(c.runner).QueryRowContext(ctx)

	var answersCount int
	if err := countRow.Scan(&answersCount); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, xerrors.Errorf("count all answers: %w", err)
	}

	query := buildLogQuery(req, &options,
		"at.try_time",
		"tg.id",
		"tg.name",
		"t.id",
		"t.name",
		"tm.id",
		"tm.name",
		"u.id",
		"u.username",
		"at.accepted",
		"at.answer",
	)
	if options.PageToken != nil && !options.DateDesc {
		query = query.Where("extract(epoch from at.try_time)*1000 > ?", *options.PageToken)
	} else if options.PageToken != nil && options.DateDesc {
		query = query.Where("extract(epoch from at.try_time)*1000 < ?", *options.PageToken)
	} else if options.PageNumber != nil {
		query = query.Offset(uint64(options.PageSize * *options.PageNumber))
	}
	query = query.Limit(uint64(options.PageSize))

	rows, err := query.RunWith(c.runner).QueryContext(ctx)
	if err != nil {
		return nil, xerrors.Errorf("query answers: %w", err)
	}
	defer func() { _ = rows.Close() }()

	answerLogs := make([]storage.AnswerLog, 0, options.PageSize)
	for rows.Next() {
		var userName, userID sql.NullString
		al := storage.AnswerLog{
			TaskGroup: &storage.TaskGroup{},
			Task:      &storage.Task{},
			Team:      &storage.Team{},
		}
		if err = rows.Scan(
			&al.AnswerTime,
			&al.TaskGroup.ID,
			&al.TaskGroup.Name,
			&al.Task.ID,
			&al.Task.Name,
			&al.Team.ID,
			&al.Team.Name,
			&userID,
			&userName,
			&al.Accepted,
			&al.Answer,
		); err != nil {
			return nil, xerrors.Errorf("scan row: %w", err)
		}
		if userName.Valid && userID.Valid {
			al.User = &storage.User{
				ID:       storage.ID(userID.String),
				Username: userName.String,
			}
		}

		answerLogs = append(answerLogs, al)
	}
	if err = rows.Err(); err != nil {
		return nil, xerrors.Errorf("iter rows: %w", err)
	}
	pagesNum := answersCount / options.PageSize
	if answersCount%options.PageSize > 0 {
		pagesNum++
	}

	var nexToken int64
	if len(answerLogs) > 0 {
		last := answerLogs[len(answerLogs)-1]
		nexToken = last.AnswerTime.UnixMilli()
	}

	res := &storage.AnswerLogRecords{
		AnswerLogs: answerLogs,
		TotalPages: pagesNum,
		NextToken:  nexToken,
	}

	return res, nil
}

func (c *Client) GetPenalties(ctx context.Context, req *storage.GetPenaltiesRequest) (storage.TeamPenalties, error) {
	query := sq.Select("p.team_id", "p.value").
		From("questspace.team_penalty p").
		PlaceholderFormat(sq.Dollar)
	if len(req.TeamIDs) > 0 {
		query = query.Where(sq.Eq{"p.team_id": req.TeamIDs})
	}
	if req.QuestID != "" {
		query = query.LeftJoin("questspace.team t ON t.id = p.team_id").Where(sq.Eq{"t.quest_id": req.QuestID})
	}

	rows, err := query.RunWith(c.runner).QueryContext(ctx)
	if err != nil {
		return nil, xerrors.Errorf("query rows: %w", err)
	}
	defer func() { _ = rows.Close() }()

	res := make(storage.TeamPenalties)
	for rows.Next() {
		var p storage.Penalty
		if err = rows.Scan(&p.TeamID, &p.Value); err != nil {
			return nil, xerrors.Errorf("scan row: %w", err)
		}
		vals := res[p.TeamID]
		vals = append(vals, p)
		res[p.TeamID] = vals
	}
	if err := rows.Err(); err != nil {
		return nil, xerrors.Errorf("iter rows: %w", err)
	}

	return res, nil
}

func (c *Client) CreatePenalty(ctx context.Context, req *storage.CreatePenaltyRequest) error {
	var err error
	// TODO(svayp11): Refactor this abomination
	deleteQuery := `DELETE FROM questspace.team_penalty WHERE team_id = $1`
	if _, err = c.runner.ExecContext(ctx, deleteQuery, req.TeamID); err != nil {
		return xerrors.Errorf("delete all previous penalties: %w", err)
	}

	insertQuery := `INSERT INTO questspace.team_penalty (team_id, value) VALUES ($1, $2)`
	if _, err = c.runner.ExecContext(ctx, insertQuery, req.TeamID, req.Penalty); err != nil {
		return xerrors.Errorf("add new penalty: %w", err)
	}
	return nil
}
