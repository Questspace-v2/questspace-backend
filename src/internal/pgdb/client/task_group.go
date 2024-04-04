package pgdb

import (
	"context"

	sq "github.com/Masterminds/squirrel"
	"golang.org/x/xerrors"

	"questspace/pkg/storage"
)

var _ storage.TaskGroupStorage = &Client{}

func (c *Client) CreateTaskGroup(ctx context.Context, req *storage.CreateTaskGroupRequest) (*storage.TaskGroup, error) {
	values := []interface{}{req.Name, req.OrderIdx, req.QuestID}
	query := sq.Insert("questspace.task_group").
		Columns("name", "order_idx", "quest_id").
		Suffix("RETURNING id").
		PlaceholderFormat(sq.Dollar)
	if req.PubTime != nil {
		values = append(values, req.PubTime)
		query = query.Columns("pub_time")
	}
	query = query.Values(values...)

	row := query.RunWith(c.runner).QueryRowContext(ctx)
	taskGroup := storage.TaskGroup{
		Name:     req.Name,
		OrderIdx: req.OrderIdx,
		Quest:    &storage.Quest{ID: req.QuestID},
		PubTime:  req.PubTime,
	}
	if err := row.Scan(&taskGroup.ID); err != nil {
		return nil, xerrors.Errorf("scan row: %w", err)
	}

	return &taskGroup, nil
}

func (c *Client) GetTaskGroup(ctx context.Context, req *storage.GetTaskGroupRequest) (*storage.TaskGroup, error) {
	query := sq.Select("id", "name", "order_idx", "pub_time", "quest_id").
		From("questspace.task_group").
		Where(sq.Eq{"id": req.ID}).
		PlaceholderFormat(sq.Dollar)
	row := query.RunWith(c.runner).QueryRowContext(ctx)

	taskGroup := storage.TaskGroup{Quest: &storage.Quest{}}
	if err := row.Scan(&taskGroup.ID, &taskGroup.Name, &taskGroup.OrderIdx, &taskGroup.PubTime, &taskGroup.Quest.ID); err != nil {
		return nil, xerrors.Errorf("scan row: %w", err)
	}
	if req.IncludeTasks {
		tasks, err := c.GetTasks(ctx, &storage.GetTasksRequest{GroupIDs: []string{req.ID}})
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
	query := sq.Select("id", "name", "order_idx", "pub_time").
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
	var groupIDs []string
	for rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, xerrors.Errorf("iter rows: %w", err)
		}
		taskGroup := storage.TaskGroup{Quest: &storage.Quest{ID: req.QuestID}}
		if err := rows.Scan(&taskGroup.ID, &taskGroup.Name, &taskGroup.OrderIdx, &taskGroup.PubTime); err != nil {
			return nil, xerrors.Errorf("scan row: %w", err)
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
		Suffix("RETURNING id, name, order_idx, pub_time, quest_id").
		PlaceholderFormat(sq.Dollar)
	switch {
	case req.Name != "":
		query = query.Set("name", req.Name)
		fallthrough
	case req.PubTime != nil:
		query = query.Set("pub_time", req.PubTime)
	}

	row := query.RunWith(c.runner).QueryRowContext(ctx)
	taskGroup := storage.TaskGroup{Quest: &storage.Quest{}}
	if err := row.Scan(&taskGroup.ID, &taskGroup.Name, &taskGroup.OrderIdx, &taskGroup.PubTime, &taskGroup.Quest.ID); err != nil {
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
