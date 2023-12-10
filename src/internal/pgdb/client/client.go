package pgdb

import (
	"context"
	"database/sql"
	"errors"

	"github.com/spkg/ptr"

	"github.com/jackc/pgx"

	"golang.yandex/hasql"

	"questspace/pkg/storage"

	"golang.org/x/xerrors"

	sq "github.com/Masterminds/squirrel"
)

const uniqueViolationCode = "23505"

type Client struct {
	conn *hasql.Cluster
}

func NewClient(cl *hasql.Cluster) *Client {
	return &Client{conn: cl}
}

func (c *Client) CreateUser(ctx context.Context, req *storage.CreateUserRequest) (*storage.User, error) {
	node, err := c.conn.WaitForPrimary(ctx)
	if err != nil {
		return nil, xerrors.Errorf("failed to await primary node: %w", err)
	}

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
	row := node.DB().QueryRowContext(ctx, queryStr, args...)

	var id string
	if err := row.Scan(&id); err != nil {
		pgErr := &pgx.PgError{}
		if errors.As(err, pgErr) && pgErr.Code == uniqueViolationCode {
			return nil, storage.ErrExists
		}
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
	node, err := c.conn.WaitForAlive(ctx)
	if err != nil {
		return nil, xerrors.Errorf("failed to await alive node: %w", err)
	}

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
	row := node.DB().QueryRowContext(ctx, queryStr, args...)

	user := &storage.User{}
	if err := row.Scan(&user.Id, &user.Username, &user.AvatarURL); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, xerrors.Errorf("failed to scan row: %w", err)
	}
	return user, nil
}

func (c *Client) UpdateUser(ctx context.Context, req *storage.UpdateUserRequest) (*storage.User, error) {
	node, err := c.conn.WaitForPrimary(ctx)
	if err != nil {
		return nil, xerrors.Errorf("failed to await primary node: %w", err)
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

	queryStr, args, err := query.ToSql()
	if err != nil {
		return nil, xerrors.Errorf("failed to build query string: %w", err)
	}
	rows := node.DB().QueryRowContext(ctx, queryStr, args...)

	user := &storage.User{}
	if err := rows.Scan(&user.Id, &user.Username, &user.AvatarURL); err != nil {
		return nil, xerrors.Errorf("failed to scan row: %w", err)
	}

	return user, nil
}

func (c *Client) GetUserPasswordHash(ctx context.Context, req *storage.GetUserRequest) (string, error) {
	node, err := c.conn.WaitForAlive(ctx)
	if err != nil {
		return "", xerrors.Errorf("failed to await alive node: %w", err)
	}

	query := sq.Select("password").
		From("questspace.user").
		PlaceholderFormat(sq.Dollar)
	if req.Id != "" {
		query = query.Where(sq.Eq{"id": req.Id})
	} else if req.Username != "" {
		query = query.Where(sq.Eq{"username": req.Username})
	} else {
		return "", xerrors.New("either user id or username should be present")
	}

	queryStr, args, err := query.ToSql()
	if err != nil {
		return "", xerrors.Errorf("failed to build query string: %w", err)
	}
	row := node.DB().QueryRowContext(ctx, queryStr, args...)
	var pw []byte

	if err := row.Scan(&pw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", storage.ErrNotFound
		}
		return "", xerrors.Errorf("failed to scan row: %w", err)
	}

	return string(pw), nil
}

func (c *Client) CreateQuest(ctx context.Context, req *storage.CreateQuestRequest) (*storage.Quest, error) {
	node, err := c.conn.WaitForPrimary(ctx)
	if err != nil {
		return nil, xerrors.Errorf("failed to await primary node: %w", err)
	}

	query := sq.
		Insert("questspace.quest").
		Columns("name", "description", "access", "creator_name", "registration_deadline",
			"start_time", "finish_time", "media_link", "max_team_cap").
		Suffix("RETURNING id").
		Values(req.Name, req.Description, req.Access, req.CreatorName,
			req.RegistrationDeadline, req.StartTime, req.FinishTime, req.MediaLink, req.MaxTeamCap).
		PlaceholderFormat(sq.Dollar)

	queryStr, args, err := query.ToSql()
	if err != nil {
		return nil, xerrors.Errorf("failed to build query string: %w", err)
	}
	row := node.DB().QueryRowContext(ctx, queryStr, args...)

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
	if err != nil {
		return nil, xerrors.Errorf("failed to get quest creator: %w", err)
	}
	return &quest, nil
}

func (c *Client) GetQuest(ctx context.Context, req *storage.GetQuestRequest) (*storage.Quest, error) {
	node, err := c.conn.WaitForAlive(ctx)
	if err != nil {
		return nil, xerrors.Errorf("failed to await db readiness: %w", err)
	}

	query := sq.
		Select("q.id", "q.name", "q.description", "q.access", "q.avatar_url", "q.registration_deadline",
			"q.start_time", "q.finish_time", "q.media_link", "q.max_team_cap", "u.id", "u.username", "u.avatar_url").
		From("questspace.q q").
		Where(sq.Eq{"id": req.Id}).
		LeftJoin("questspace.user u on u.username = q.creator_name").
		PlaceholderFormat(sq.Dollar)

	queryStr, args, err := query.ToSql()
	if err != nil {
		return nil, xerrors.Errorf("failed to build query string: %w", err)
	}
	row := node.DB().QueryRowContext(ctx, queryStr, args...)

	q := &storage.Quest{Creator: &storage.User{}}
	var (
		finishTime sql.NullTime
		maxTeamCap sql.NullInt32
	)
	if err := row.Scan(&q.Id, &q.Name, &q.Description, &q.Creator.AvatarURL, &q.RegistrationDeadline,
		&q.StartTime, &finishTime, &q.MediaLink, &maxTeamCap, &q.Creator.Id, &q.Creator.Username, &q.Creator.AvatarURL); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, xerrors.Errorf("failed to scan row: %w", err)
	}
	if finishTime.Valid {
		q.FinishTime = ptr.Time(finishTime.Time)
	}
	if maxTeamCap.Valid {
		q.MaxTeamCap = ptr.Int(int(maxTeamCap.Int32))
	}
	return q, nil
}

func (c *Client) UpdateQuest(ctx context.Context, req *storage.UpdateQuestRequest) (*storage.Quest, error) {
	node, err := c.conn.WaitForPrimary(ctx)
	if err != nil {
		return nil, xerrors.Errorf("failed to await primary node: %w", err)
	}

	query := sq.
		Update("questspace.quest").
		Where(sq.Eq{"id": req.Id}).
		Suffix("RETURNING id, name, description, access, creator_name, " +
			"registration_deadline, start_time, finish_time, media_link, max_team_cap").
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
	row := node.DB().QueryRowContext(ctx, queryStr, args...)

	quest := &storage.Quest{}
	var (
		finishTime sql.NullTime
		maxTeamCap sql.NullInt32
	)
	var creatorName string
	if err := row.Scan(&quest.Id, &quest.Name, &quest.Description, &creatorName, &quest.RegistrationDeadline,
		&quest.StartTime, &finishTime, &quest.MediaLink, &maxTeamCap); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, xerrors.Errorf("failed to scan row: %w", err)
	}
	if finishTime.Valid {
		quest.FinishTime = ptr.Time(finishTime.Time)
	}
	if maxTeamCap.Valid {
		quest.MaxTeamCap = ptr.Int(int(maxTeamCap.Int32))
	}
	quest.Creator, err = c.GetUser(ctx, &storage.GetUserRequest{Username: creatorName})
	return quest, nil
}
