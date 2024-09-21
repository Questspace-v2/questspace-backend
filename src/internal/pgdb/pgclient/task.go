package pgclient

import (
	"context"
	"database/sql"
	"errors"
	"slices"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/xerrors"

	"questspace/pkg/storage"
)

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
		Group:          &storage.TaskGroup{ID: req.GroupID},
		Name:           req.Name,
		OrderIdx:       req.OrderIdx,
		Question:       req.Question,
		Reward:         req.Reward,
		CorrectAnswers: slices.Clone(req.CorrectAnswers),
		Verification:   req.Verification,
		Hints:          slices.Clone(req.Hints),
		MediaLinks:     req.MediaLinks,
		MediaLink:      req.MediaLink,
		PubTime:        req.PubTime,
	}
	if err := row.Scan(&task.ID); err != nil {
		return nil, xerrors.Errorf("scan row: %w", err)
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
	pub_time
FROM questspace.task
	WHERE id = $1
`

func (c *Client) GetTask(ctx context.Context, req *storage.GetTaskRequest) (*storage.Task, error) {
	row := c.runner.QueryRowContext(ctx, getTaskQuery, req.ID)
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

	return &task, nil
}

func (c *Client) GetAnswerData(ctx context.Context, req *storage.GetTaskRequest) (*storage.Task, error) {
	query := sq.Select("correct_answers", "reward", "verification", "hints").
		From("questspace.task").
		Where(sq.Eq{"id": req.ID}).
		PlaceholderFormat(sq.Dollar)

	row := query.RunWith(c.runner).QueryRowContext(ctx)
	task := storage.Task{ID: req.ID}
	pgMap := pgtype.NewMap()
	if err := row.Scan(pgMap.SQLScanner(&task.CorrectAnswers), &task.Reward, &task.Verification, pgMap.SQLScanner(&task.Hints)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, xerrors.Errorf("scan row: %w", err)
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
	tasks := make(map[string][]storage.Task)
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

		group := tasks[task.Group.ID]
		group = append(group, task)
		tasks[task.Group.ID] = group
	}
	if err := rows.Err(); err != nil {
		return nil, xerrors.Errorf("iter rows: %w", err)
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
	}
	if len(req.MediaLink) > 0 {
		query = query.Set("media_url", req.MediaLink)
	}
	if len(req.MediaLinks) > 0 {
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
