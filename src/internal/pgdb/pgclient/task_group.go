package pgclient

import (
	"context"
	"database/sql"
	"net/http"

	sq "github.com/Masterminds/squirrel"
	"golang.org/x/xerrors"

	"questspace/pkg/httperrors"
	"questspace/pkg/storage"
)

var _ storage.TaskGroupStorage = &Client{}

func (c *Client) CreateTaskGroup(ctx context.Context, req *storage.CreateTaskGroupRequest) (*storage.TaskGroup, error) {
	values := []interface{}{req.Name, req.OrderIdx, req.Sticky, req.QuestID, req.HasTimeLimit}
	query := sq.Insert("questspace.task_group").
		Columns("name", "order_idx", "sticky", "quest_id", "has_time_limit").
		Suffix("RETURNING id").
		PlaceholderFormat(sq.Dollar)
	if req.PubTime != nil {
		values = append(values, req.PubTime)
		query = query.Columns("pub_time")
	}
	if len(req.Description) > 0 {
		values = append(values, req.Description)
		query = query.Columns("description")
	}
	if req.TimeLimit != nil {
		values = append(values, *req.TimeLimit)
		query = query.Columns("time_limit")
	} else if req.HasTimeLimit {
		return nil, httperrors.New(http.StatusBadRequest, "task group has `has_time_limit` without actual time limit")
	}
	query = query.Values(values...)

	row := query.RunWith(c.runner).QueryRowContext(ctx)
	taskGroup := storage.TaskGroup{
		Name:         req.Name,
		Description:  req.Description,
		OrderIdx:     req.OrderIdx,
		Sticky:       req.Sticky,
		Quest:        &storage.Quest{ID: req.QuestID},
		PubTime:      req.PubTime,
		HasTimeLimit: req.HasTimeLimit,
		TimeLimit:    req.TimeLimit,
	}
	if err := row.Scan(&taskGroup.ID); err != nil {
		return nil, xerrors.Errorf("scan row: %w", err)
	}

	return &taskGroup, nil
}

func (c *Client) GetTaskGroup(ctx context.Context, req *storage.GetTaskGroupRequest) (*storage.TaskGroup, error) {
	query := `
	SELECT id, name, description, order_idx, sticky, pub_time, quest_id, has_time_limit, time_limit
	FROM questspace.task_group
	WHERE id = $1
`
	row := c.runner.QueryRowContext(ctx, query, req.ID)

	taskGroup := storage.TaskGroup{Quest: &storage.Quest{}}
	var descr sql.NullString
	if err := row.Scan(
		&taskGroup.ID,
		&taskGroup.Name,
		&descr,
		&taskGroup.OrderIdx,
		&taskGroup.Sticky,
		&taskGroup.PubTime,
		&taskGroup.Quest.ID,
		&taskGroup.HasTimeLimit,
		&taskGroup.TimeLimit,
	); err != nil {
		return nil, xerrors.Errorf("scan row: %w", err)
	}
	if descr.Valid {
		taskGroup.Description = descr.String
	}
	if req.IncludeTasks {
		tasks, err := c.GetTasks(ctx, &storage.GetTasksRequest{GroupIDs: []storage.ID{req.ID}})
		if err != nil {
			return nil, xerrors.Errorf("get tasks: %w", err)
		}
		taskGroup.Tasks = tasks[req.ID]
		for i := range len(taskGroup.Tasks) {
			taskGroup.Tasks[i].Group = &taskGroup
		}
	}

	return &taskGroup, nil
}

func (c *Client) GetTaskGroups(ctx context.Context, req *storage.GetTaskGroupsRequest) ([]storage.TaskGroup, error) {
	query := sq.Select("id", "name", "description", "order_idx", "sticky", "pub_time", "has_time_limit", "time_limit").
		From("questspace.task_group").
		Where(sq.Eq{"quest_id": req.QuestID}).
		OrderBy("order_idx").
		PlaceholderFormat(sq.Dollar)
	rows, err := query.RunWith(c.runner).QueryContext(ctx)
	if err != nil {
		return nil, xerrors.Errorf("query rows: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var taskGroups []storage.TaskGroup
	var groupIDs []storage.ID
	for rows.Next() {
		var descr sql.NullString
		if err := rows.Err(); err != nil {
			return nil, xerrors.Errorf("iter rows: %w", err)
		}
		taskGroup := storage.TaskGroup{Quest: &storage.Quest{ID: req.QuestID}}
		if err := rows.Scan(
			&taskGroup.ID,
			&taskGroup.Name,
			&descr,
			&taskGroup.OrderIdx,
			&taskGroup.Sticky,
			&taskGroup.PubTime,
			&taskGroup.HasTimeLimit,
			&taskGroup.TimeLimit,
		); err != nil {
			return nil, xerrors.Errorf("scan row: %w", err)
		}
		if descr.Valid {
			taskGroup.Description = descr.String
		}
		taskGroups = append(taskGroups, taskGroup)
		groupIDs = append(groupIDs, taskGroup.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, xerrors.Errorf("iter rows: %w", err)
	}

	if req.IncludeTasks {
		tasks, err := c.GetTasks(ctx, &storage.GetTasksRequest{GroupIDs: groupIDs})
		if err != nil {
			return nil, xerrors.Errorf("get tasks: %w", err)
		}
		for i := range len(taskGroups) {
			tg := taskGroups[i]
			taskGroups[i].Tasks = tasks[tg.ID]

			for j := range len(tg.Tasks) {
				tg.Tasks[j].Group = &tg
			}
		}
	}

	return taskGroups, nil
}

func (c *Client) UpdateTaskGroup(ctx context.Context, req *storage.UpdateTaskGroupRequest) (*storage.TaskGroup, error) {
	query := sq.Update("questspace.task_group").
		Where(sq.Eq{"id": req.ID}).
		Set("order_idx", req.OrderIdx).
		Suffix("RETURNING id, name, order_idx, pub_time, quest_id, has_time_limit, time_limit").
		PlaceholderFormat(sq.Dollar)
	if len(req.Name) > 0 {
		query = query.Set("name", req.Name)
	}
	if req.Description != nil {
		query = query.Set("description", *req.Description)
	}
	if req.PubTime != nil {
		query = query.Set("pub_time", req.PubTime)
	}
	if req.Sticky != nil {
		query = query.Set("sticky", *req.Sticky)
	}
	if req.HasTimeLimit != nil {
		query = query.Set("has_time_limit", *req.HasTimeLimit)
	}
	if req.TimeLimit != nil {
		query = query.Set("time_limit", *req.TimeLimit)
	}

	row := query.RunWith(c.runner).QueryRowContext(ctx)
	taskGroup := storage.TaskGroup{Quest: &storage.Quest{}}
	if err := row.Scan(
		&taskGroup.ID,
		&taskGroup.Name,
		&taskGroup.OrderIdx,
		&taskGroup.PubTime,
		&taskGroup.Quest.ID,
		&taskGroup.HasTimeLimit,
		&taskGroup.TimeLimit,
	); err != nil {
		return nil, xerrors.Errorf("scan row: %w", err)
	}

	return &taskGroup, nil
}

func (c *Client) DeleteTaskGroup(ctx context.Context, req *storage.DeleteTaskGroupRequest) error {
	query := sq.Delete("questspace.task_group").
		Where(sq.Eq{"id": req.ID}).
		PlaceholderFormat(sq.Dollar)
	_, err := query.RunWith(c.runner).ExecContext(ctx)
	if err != nil {
		return xerrors.Errorf("exec query: %w", err)
	}
	return nil
}
