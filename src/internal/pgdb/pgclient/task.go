package pgclient

import (
	"context"
	"database/sql"
	"errors"
	"slices"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/yandex/perforator/library/go/core/xerrors"

	"questspace/pkg/storage"
)

func (c *Client) createHints(ctx context.Context, taskID storage.ID, hints []storage.CreateHintRequest) ([]storage.Hint, error) {
	if len(hints) == 0 {
		return []storage.Hint{}, nil
	}
	hintsRes := make([]storage.Hint, 0, len(hints))
	query := sq.Insert("questspace.hint").
		Columns(
			"task_id",
			"index",
			"name",
			"text",
			"penalty_percent",
			"penalty_score",
		).
		PlaceholderFormat(sq.Dollar)

	for i, hintReq := range hints {
		hintArgs := []any{
			taskID,
			i,
			hintReq.Name,
			hintReq.Text,
			hintReq.Penalty.PercentOpt(),
			hintReq.Penalty.ScoreOpt(),
		}
		query = query.Values(hintArgs...)

		if hintReq.Penalty.PercentOpt() == nil && hintReq.Penalty.ScoreOpt() == nil {
			return nil, xerrors.Errorf("both opts of #%d hint penalty are empty", i)
		}
		if hintReq.Penalty.PercentOpt() != nil && hintReq.Penalty.ScoreOpt() != nil {
			return nil, xerrors.Errorf("both opts of #%d hint penalty are not empty", i)
		}

		hintsRes = append(hintsRes, storage.Hint{
			TaskID:  taskID,
			Index:   i,
			Name:    hintReq.Name,
			Text:    hintReq.Text,
			Penalty: hintReq.Penalty,
		})
	}

	_, err := query.RunWith(c.runner).ExecContext(ctx)
	if err != nil {
		return nil, xerrors.Errorf("exec query: %w", err)
	}

	return hintsRes, nil
}

func (c *Client) getHints(ctx context.Context, taskIDs []storage.ID) (hintsByTaskID map[storage.ID][]storage.Hint, err error) {
	query := sq.Select(
		"task_id",
		"index",
		"name",
		"text",
		"penalty_percent",
		"penalty_score",
	).From("questspace.hint").
		Where(sq.Eq{"task_id": taskIDs}).
		OrderBy("index").
		PlaceholderFormat(sq.Dollar)

	rows, err := query.RunWith(c.runner).QueryContext(ctx)
	if err != nil {
		return nil, xerrors.Errorf("query rows: %w", err)
	}
	defer func() { _ = rows.Close() }()

	hintsByTaskID = make(map[storage.ID][]storage.Hint, len(taskIDs))
	for rows.Next() {
		var hint storage.Hint
		var percent, score *int
		if err = rows.Scan(
			&hint.TaskID,
			&hint.Index,
			&hint.Name,
			&hint.Text,
			&percent,
			&score,
		); err != nil {
			return nil, xerrors.Errorf("scan row: %w", err)
		}
		if percent != nil {
			hint.Penalty, err = storage.NewPercentagePenalty(*percent)
			if err != nil {
				return nil, xerrors.Errorf("bad penalty: %w", err)
			}
		}
		if score != nil {
			hint.Penalty = storage.NewScorePenalty(*score)
		}

		taskHints := hintsByTaskID[hint.TaskID]
		taskHints = append(taskHints, hint)
		hintsByTaskID[hint.TaskID] = taskHints
	}
	if err = rows.Err(); err != nil {
		return nil, xerrors.Errorf("iter rows: %w", err)
	}

	return hintsByTaskID, nil
}

func (c *Client) updateHints(ctx context.Context, taskID storage.ID, hints []storage.CreateHintRequest) ([]storage.Hint, error) {
	const query = `DELETE FROM questspace.hint WHERE task_id = $1`
	_, err := c.runner.ExecContext(ctx, query, taskID)
	if err != nil {
		return nil, xerrors.Errorf("delete previous hints: %w", err)
	}

	return c.createHints(ctx, taskID, hints)
}

func (c *Client) CreateTask(ctx context.Context, req *storage.CreateTaskRequest) (*storage.Task, error) {
	values := []any{
		req.OrderIdx,
		req.GroupID,
		req.Name,
		req.Question,
		req.Reward,
		pgtype.FlatArray[string](req.CorrectAnswers),
		req.Verification,
		pgtype.FlatArray[string](req.Hints),
		req.MediaLink,
	}

	query := sq.Insert("questspace.task").
		Columns(
			"order_idx",
			"group_id",
			"name",
			"question",
			"reward",
			"correct_answers",
			"verification",
			"hints",
			"media_url",
		).
		Suffix("RETURNING id").
		PlaceholderFormat(sq.Dollar)
	if len(req.MediaLinks) == 0 && len(req.MediaLink) > 0 {
		req.MediaLinks = []string{req.MediaLink}
	}
	if len(req.MediaLinks) > 0 {
		query = query.Columns("media_urls")
		values = append(values, pgtype.FlatArray[string](req.MediaLinks))
	}
	if req.PubTime != nil {
		query = query.Columns("pub_time")
		values = append(values, req.PubTime)
	}
	query = query.Values(values...)

	row := query.RunWith(c.runner).QueryRowContext(ctx)

	task := storage.Task{
		Group:           &storage.TaskGroup{ID: req.GroupID},
		Name:            req.Name,
		OrderIdx:        req.OrderIdx,
		Question:        req.Question,
		Reward:          req.Reward,
		CorrectAnswers:  slices.Clone(req.CorrectAnswers),
		Verification:    req.Verification,
		VerificationNew: req.Verification,
		Hints:           append([]string{}, req.Hints...),
		FullHints:       []storage.Hint{},
		MediaLinks:      req.MediaLinks,
		MediaLink:       req.MediaLink,
		PubTime:         req.PubTime,
	}
	if err := row.Scan(&task.ID); err != nil {
		return nil, xerrors.Errorf("scan row: %w", err)
	}

	if len(req.FullHints) == 0 {
		for _, hintText := range task.Hints {
			req.FullHints = append(req.FullHints, storage.CreateHintRequest{Text: hintText, Penalty: storage.DefaultPenalty})
		}
	}
	var err error
	task.FullHints, err = c.createHints(ctx, task.ID, req.FullHints)
	if err != nil {
		return nil, xerrors.Errorf("create hints: %w", err)
	}

	return &task, nil
}

const getTaskQuery = `
SELECT
	order_idx,
	name,
	question,
	reward,
	correct_answers,
	verification,
	hints,
	media_url,
	media_urls,
	pub_time,
	group_id
FROM questspace.task
	WHERE id = $1
`

func (c *Client) GetTask(ctx context.Context, req *storage.GetTaskRequest) (*storage.Task, error) {
	row := c.runner.QueryRowContext(ctx, getTaskQuery, req.ID)
	task := storage.Task{ID: req.ID, Group: &storage.TaskGroup{}}
	pgMap := pgtype.NewMap()
	if err := row.Scan(
		&task.OrderIdx,
		&task.Name,
		&task.Question,
		&task.Reward,
		pgMap.SQLScanner(&task.CorrectAnswers),
		&task.Verification,
		pgMap.SQLScanner(&task.Hints),
		&task.MediaLink,
		pgMap.SQLScanner(&task.MediaLinks),
		&task.PubTime,
		&task.Group.ID,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, xerrors.Errorf("scan row: %w", err)
	}
	if len(task.MediaLinks) == 0 && len(task.MediaLink) > 0 {
		task.MediaLinks = []string{task.MediaLink}
	}
	task.Hints = append([]string{}, task.Hints...)
	task.VerificationNew = task.Verification
	task.FullHints = []storage.Hint{}

	hintsByID, err := c.getHints(ctx, []storage.ID{req.ID})
	if err != nil {
		return nil, xerrors.Errorf("get task hints: %w", err)
	}
	if hintsByID[req.ID] != nil {
		task.FullHints = hintsByID[req.ID]
	}

	return &task, nil
}

func (c *Client) GetAnswerData(ctx context.Context, req *storage.GetTaskRequest) (*storage.Task, error) {
	query := sq.Select("group_id", "correct_answers", "reward", "verification", "hints").
		From("questspace.task").
		Where(sq.Eq{"id": req.ID}).
		PlaceholderFormat(sq.Dollar)

	row := query.RunWith(c.runner).QueryRowContext(ctx)
	task := storage.Task{ID: req.ID, Group: &storage.TaskGroup{}}
	pgMap := pgtype.NewMap()
	if err := row.Scan(
		&task.Group.ID,
		pgMap.SQLScanner(&task.CorrectAnswers),
		&task.Reward,
		&task.Verification,
		pgMap.SQLScanner(&task.Hints),
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, xerrors.Errorf("scan row: %w", err)
	}
	task.VerificationNew = task.Verification
	task.FullHints = []storage.Hint{}

	hintsByID, err := c.getHints(ctx, []storage.ID{req.ID})
	if err != nil {
		return nil, xerrors.Errorf("get task hints: %w", err)
	}
	if hintsByID[req.ID] != nil {
		task.FullHints = hintsByID[req.ID]
	}

	return &task, nil
}

func (c *Client) GetTasks(ctx context.Context, req *storage.GetTasksRequest) (storage.GetTasksResponse, error) {
	query := sq.Select(
		"t.id",
		"t.order_idx",
		"t.name",
		"t.question",
		"t.reward",
		"t.group_id",
		"t.correct_answers",
		"t.verification",
		"t.hints",
		"t.media_url",
		"t.media_urls",
		"t.pub_time",
	).
		From("questspace.task t").
		OrderBy("t.group_id", "t.order_idx ASC").
		PlaceholderFormat(sq.Dollar)
	if req.QuestID != "" {
		query = query.
			LeftJoin("questspace.task_group tg ON tg.id = t.group_id").
			Where(sq.Eq{"tg.quest_id": req.QuestID})
	}
	if len(req.GroupIDs) > 0 {
		query = query.Where(sq.Eq{"t.group_id": req.GroupIDs})
	}

	rows, err := query.RunWith(c.runner).QueryContext(ctx)
	if err != nil {
		return nil, xerrors.Errorf("query rows: %w", err)
	}
	defer func() { _ = rows.Close() }()

	pgMap := pgtype.NewMap()
	tasks := make(storage.GetTasksResponse)
	var ids []storage.ID
	for rows.Next() {
		task := storage.Task{Group: &storage.TaskGroup{}}
		if err := rows.Scan(
			&task.ID,
			&task.OrderIdx,
			&task.Name,
			&task.Question,
			&task.Reward,
			&task.Group.ID,
			pgMap.SQLScanner(&task.CorrectAnswers),
			&task.Verification,
			pgMap.SQLScanner(&task.Hints),
			&task.MediaLink,
			pgMap.SQLScanner(&task.MediaLinks),
			&task.PubTime,
		); err != nil {
			return nil, xerrors.Errorf("scan row: %w", err)
		}
		if len(task.MediaLinks) == 0 && len(task.MediaLink) > 0 {
			task.MediaLinks = []string{task.MediaLink}
		}
		task.VerificationNew = task.Verification
		task.Hints = append([]string{}, task.Hints...)
		task.FullHints = []storage.Hint{}

		group := tasks[task.Group.ID]
		group = append(group, task)
		tasks[task.Group.ID] = group
		ids = append(ids, task.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, xerrors.Errorf("iter rows: %w", err)
	}

	hintsByID, err := c.getHints(ctx, ids)
	if err != nil {
		return nil, xerrors.Errorf("get task hints: %w", err)
	}
	for _, tgTasks := range tasks {
		for i, tgTask := range tgTasks {
			if hintsByID[tgTask.ID] == nil {
				continue
			}
			tgTask.FullHints = hintsByID[tgTask.ID]
			tgTasks[i] = tgTask
		}
	}

	return tasks, nil
}

// UpdateTask godoc
// TODO: unit-tests
func (c *Client) UpdateTask(ctx context.Context, req *storage.UpdateTaskRequest) (*storage.Task, error) {
	query := sq.Update("questspace.task").
		Set("order_idx", req.OrderIdx).
		Where(sq.Eq{"id": req.ID}).
		Suffix(`RETURNING 
			order_idx, 
			name, 
			question, 
			reward, 
			correct_answers, 
			verification, 
			hints, 
			media_url, 
			media_urls, 
			pub_time`).
		PlaceholderFormat(sq.Dollar)
	if len(req.Name) > 0 {
		query = query.Set("name", req.Name)
	}
	if len(req.Question) > 0 {
		query = query.Set("question", req.Question)
	}
	if req.Reward != 0 {
		query = query.Set("reward", req.Reward)
	}
	if len(req.CorrectAnswers) > 0 {
		query = query.Set("correct_answers", pgtype.FlatArray[string](req.CorrectAnswers))
	}
	if len(req.Hints) > 0 {
		query = query.Set("hints", pgtype.FlatArray[string](req.Hints))

		if req.FullHints == nil {
			fullHints := make([]storage.CreateHintRequest, 0, len(req.Hints))
			for _, hintText := range req.Hints {
				fullHints = append(fullHints, storage.CreateHintRequest{Text: hintText, Penalty: storage.DefaultPenalty})
			}
			req.FullHints = &fullHints
		}
	}
	if req.MediaLink != nil {
		query = query.Set("media_url", *req.MediaLink)
	}
	if req.MediaLinks != nil {
		query = query.Set("media_urls", pgtype.FlatArray[string](req.MediaLinks))
	}
	if req.PubTime != nil {
		query = query.Set("pub_time", req.PubTime)
	}

	row := query.RunWith(c.runner).QueryRowContext(ctx)
	task := storage.Task{ID: req.ID}
	pgMap := pgtype.NewMap()
	if err := row.Scan(
		&task.OrderIdx,
		&task.Name,
		&task.Question,
		&task.Reward,
		pgMap.SQLScanner(&task.CorrectAnswers),
		&task.Verification,
		pgMap.SQLScanner(&task.Hints),
		&task.MediaLink,
		pgMap.SQLScanner(&task.MediaLinks),
		&task.PubTime,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, xerrors.Errorf("scan row: %w", err)
	}
	if len(task.MediaLinks) == 0 && len(task.MediaLink) > 0 {
		task.MediaLinks = []string{task.MediaLink}
	}
	task.Hints = append([]string{}, task.Hints...)
	task.VerificationNew = task.Verification
	task.FullHints = []storage.Hint{}

	if req.FullHints != nil {
		var err error
		task.FullHints, err = c.updateHints(ctx, task.ID, *req.FullHints)
		if err != nil {
			return nil, xerrors.Errorf("update hints: %w", err)
		}
	}

	return &task, nil
}

// DeleteTask godoc
// TODO: unit-tests
func (c *Client) DeleteTask(ctx context.Context, req *storage.DeleteTaskRequest) error {
	query := sq.Delete("questspace.task").
		Where(sq.Eq{"id": req.ID}).
		PlaceholderFormat(sq.Dollar)

	if _, err := query.RunWith(c.runner).ExecContext(ctx); err != nil {
		return xerrors.Errorf("exec query: %w", err)
	}
	return nil
}
