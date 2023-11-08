package pgdb

import (
	"context"
	"database/sql"
	"errors"

	"questspace/pkg/storage"

	"golang.org/x/xerrors"

	sq "github.com/Masterminds/squirrel"
)

type Client struct {
	db *sql.DB
}

func NewClient(conn *sql.DB) *Client {
	return &Client{db: conn}
}

func (c *Client) CreateUser(ctx context.Context, req *storage.CreateUserRequest) (*storage.User, error) {
	_, err := c.GetUser(ctx, &storage.GetUserRequest{Username: req.Username})
	if err == nil {
		return nil, storage.ErrExists
	}
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return nil, xerrors.Errorf("failed to get user: %w", err)
	}
	node, err := c.db.Conn(ctx)
	if err != nil {
		return nil, xerrors.Errorf("failed to await db readiness: %w", err)
	}
	defer func() { _ = node.Close() }()

	values := []interface{}{req.Username, []byte(req.Password)}
	query := sq.
		Insert("questspace.user").
		Columns("username", "password").
		Suffix("RETURNING id").
		PlaceholderFormat(sq.Dollar)
	if req.AvatarURL != "" {
		query = query.Columns("avatar_url")
		values = append(values, req.AvatarURL)
	}

	query = query.Values(values...)
	queryStr, args, err := query.ToSql()
	if err != nil {
		return nil, xerrors.Errorf("failed to build query string: %w", err)
	}
	row, err := node.QueryContext(ctx, queryStr, args...)
	if err != nil {
		return nil, xerrors.Errorf("failed to execute query %s: %w", queryStr, err)
	}
	defer func() { _ = row.Close() }()
	if !row.Next() {
		return nil, xerrors.Errorf("failed to get user: %w", row.Err())
	}

	var id string
	if err := row.Scan(&id); err != nil {
		return nil, xerrors.Errorf("failed to scan row: %w", err)
	}
	user := storage.User{
		Id:        id,
		Username:  req.Username,
		Password:  req.Password,
		AvatarURL: req.AvatarURL,
	}
	user.Id = id
	return &user, nil
}

func (c *Client) GetUser(ctx context.Context, req *storage.GetUserRequest) (*storage.User, error) {
	node, err := c.db.Conn(ctx)
	if err != nil {
		return nil, xerrors.Errorf("failed to await db readiness: %w", err)
	}
	defer func() { _ = node.Close() }()

	query := sq.
		Select("id", "username", "avatar_url").
		From("questspace.user").
		PlaceholderFormat(sq.Dollar)
	if req.Id != "" {
		query = query.Where(sq.Eq{"id": req.Id})
	} else if req.Username != "" {
		query = query.Where(sq.Eq{"username": req.Username})
	} else {
		return nil, xerrors.New("at least one of request fields must not be empty")
	}

	queryStr, args, err := query.ToSql()
	if err != nil {
		return nil, xerrors.Errorf("failed to build query string: %w", err)
	}
	row, err := node.QueryContext(ctx, queryStr, args...)
	if err != nil {
		return nil, xerrors.Errorf("failed to execute query %s: %w", queryStr, err)
	}
	defer func() { _ = row.Close() }()
	if !row.Next() {
		return nil, storage.ErrNotFound
	}

	user := &storage.User{}
	if err := row.Scan(&user.Id, &user.Username, &user.AvatarURL); err != nil {
		return nil, xerrors.Errorf("failed to scan row: %w", err)
	}
	return user, nil
}

func (c *Client) UpdateUser(ctx context.Context, req *storage.UpdateUserRequest) (*storage.User, error) {
	node, err := c.db.Conn(ctx)
	if err != nil {
		return nil, xerrors.Errorf("failed to await db readiness: %w", err)
	}
	defer func() { _ = node.Close() }()

	pwQuery := sq.
		Select("password").
		From("questspace.user").
		Where(sq.Eq{"id": req.Id}).
		PlaceholderFormat(sq.Dollar)
	queryStr, args, err := pwQuery.ToSql()
	if err != nil {
		return nil, xerrors.Errorf("failed to build query string: %w", err)
	}
	row := node.QueryRowContext(ctx, queryStr, args...)
	var oldPassword []byte
	if err := row.Scan(&oldPassword); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, xerrors.Errorf("failed to get stored password: %w", err)
	}
	// TODO(svayp11): Use more suitable error
	if string(oldPassword) != req.OldPassword {
		return nil, storage.ErrExists
	}

	if req.Username == "" && req.NewPassword == "" && req.AvatarURL == "" {
		return c.GetUser(ctx, &storage.GetUserRequest{Id: req.Id})
	}

	query := sq.
		Update("questspace.user").
		Where(sq.Eq{"id": req.Id}).
		Suffix("RETURNING id, username, avatar_url").
		PlaceholderFormat(sq.Dollar)
	if req.Username != "" {
		query = query.Set("username", req.Username)
	}
	if req.NewPassword != "" {
		query = query.Set("password", []byte(req.NewPassword))
	}
	if req.AvatarURL != "" {
		query = query.Set("avatar_url", req.AvatarURL)
	}

	queryStr, args, err = query.ToSql()
	if err != nil {
		return nil, xerrors.Errorf("failed to build query string: %w", err)
	}
	rows, err := node.QueryContext(ctx, queryStr, args...)
	if err != nil {
		return nil, xerrors.Errorf("failed to execute query %s: %w", queryStr, err)
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		return nil, xerrors.Errorf("failed to insert row: %w", rows.Err())
	}

	user := &storage.User{}
	if err := rows.Scan(&user.Id, &user.Username, &user.AvatarURL); err != nil {
		return nil, xerrors.Errorf("failed to scan row: %w", err)
	}
	return user, nil
}

func (c *Client) CreateQuest(ctx context.Context, req *storage.CreateQuestRequest) (*storage.Quest, error) {
	node, err := c.db.Conn(ctx)
	if err != nil {
		return nil, xerrors.Errorf("failed to await db readiness: %w", err)
	}
	defer func() { _ = node.Close() }()

	query := sq.
		Insert("questspace.quest").
		Columns("name", "description", "access", "creator_name", "registration_deadline", "start_time", "finish_time", "media_link", "max_team_cap").
		Suffix("RETURNING id").
		PlaceholderFormat(sq.Dollar)
	values := []interface{}{req.Name, req.Description, req.Access, req.CreatorName, req.RegistrationDeadline, req.StartTime, req.FinishTime, req.MediaLink, req.MaxTeamCap}
	query = query.Values(values...)

	queryStr, args, err := query.ToSql()
	if err != nil {
		return nil, xerrors.Errorf("failed to build query string: %w", err)
	}
	row, err := node.QueryContext(ctx, queryStr, args...)
	if err != nil {
		return nil, xerrors.Errorf("failed to execute query %s: %w", queryStr, err)
	}
	defer func() { _ = row.Close() }()
	if !row.Next() {
		return nil, xerrors.Errorf("failed to get user: %w", row.Err())
	}

	var id string
	if err := row.Scan(&id); err != nil {
		return nil, xerrors.Errorf("failed to scan row: %w", err)
	}
	quest := storage.Quest{
		Id:                   id,
		Name:                 req.Name,
		Description:          req.Description,
		Access:               req.Access,
		RegistrationDeadline: req.RegistrationDeadline,
		StartTime:            req.StartTime,
		FinishTime:           req.FinishTime,
		MediaLink:            req.MediaLink,
		MaxTeamCap:           req.MaxTeamCap,
	}
	quest.Creator, err = c.GetUser(ctx, &storage.GetUserRequest{Username: req.CreatorName})
	return &quest, nil
}

func (c *Client) GetQuest(ctx context.Context, req *storage.GetQuestRequest) (*storage.Quest, error) {
	node, err := c.db.Conn(ctx)
	if err != nil {
		return nil, xerrors.Errorf("failed to await db readiness: %w", err)
	}
	defer func() { _ = node.Close() }()

	query := sq.
		Select("id", "name", "description", "access", "creator_name", "registration_deadline", "start_time", "finish_time", "media_link", "max_team_cap").
		From("questspace.quest").
		Where(sq.Eq{"id": req.Id}).
		PlaceholderFormat(sq.Dollar)

	queryStr, args, err := query.ToSql()
	if err != nil {
		return nil, xerrors.Errorf("failed to build query string: %w", err)
	}
	row, err := node.QueryContext(ctx, queryStr, args...)
	if err != nil {
		return nil, xerrors.Errorf("failed to execute query %s: %w", queryStr, err)
	}
	defer func() { _ = row.Close() }()
	if !row.Next() {
		return nil, storage.ErrNotFound
	}

	quest := &storage.Quest{}
	var creatorName string
	if err := row.Scan(&quest.Id, &quest.Name, &quest.Description, &creatorName, &quest.RegistrationDeadline, &quest.StartTime, &quest.FinishTime, &quest.MediaLink, &quest.MaxTeamCap); err != nil {
		return nil, xerrors.Errorf("failed to scan row: %w", err)
	}
	quest.Creator, err = c.GetUser(ctx, &storage.GetUserRequest{Username: creatorName})
	return quest, nil
}

func (c *Client) UpdateQuest(ctx context.Context, req *storage.UpdateQuestRequest) (*storage.Quest, error) {
	node, err := c.db.Conn(ctx)
	if err != nil {
		return nil, xerrors.Errorf("failed to await db readiness: %w", err)
	}
	defer func() { _ = node.Close() }()

	query := sq.
		Update("questspace.quest").
		Where(sq.Eq{"id": req.Id}).
		Suffix("RETURNING id, name, description, access, creator_name, registration_deadline, start_time, finish_time, media_link, max_team_cap").
		PlaceholderFormat(sq.Dollar)
	if req.Name != "" {
		query = query.Set("name", req.Name)
	}
	if req.Description != "" {
		query = query.Set("description", req.Description)
	}
	if req.Access != "" {
		query = query.Set("access", req.Access)
	}
	if req.CreatorName != "" {
		query = query.Set("creator_name", req.CreatorName)
	}
	if req.RegistrationDeadline != nil {
		query = query.Set("registration_deadline", req.RegistrationDeadline)
	}
	if req.StartTime != nil {
		query = query.Set("start_time", req.StartTime)
	}
	if req.FinishTime != nil {
		query = query.Set("finish_time", req.FinishTime)
	}
	if req.MaxTeamCap != nil {
		query = query.Set("max_team_cap", req.MaxTeamCap)
	}

	queryStr, args, err := query.ToSql()
	if err != nil {
		return nil, xerrors.Errorf("failed to build query string: %w", err)
	}
	row, err := node.QueryContext(ctx, queryStr, args...)
	if err != nil {
		return nil, xerrors.Errorf("failed to execute query %s: %w", queryStr, err)
	}
	defer func() { _ = row.Close() }()
	if !row.Next() {
		return nil, xerrors.Errorf("failed to insert row: %w", row.Err())
	}

	quest := &storage.Quest{}
	var creatorName string
	if err := row.Scan(&quest.Id, &quest.Name, &quest.Description, &creatorName, &quest.RegistrationDeadline, &quest.StartTime, &quest.FinishTime, &quest.MediaLink, &quest.MaxTeamCap); err != nil {
		return nil, xerrors.Errorf("failed to scan row: %w", err)
	}
	quest.Creator, err = c.GetUser(ctx, &storage.GetUserRequest{Username: creatorName})
	return quest, nil
}
