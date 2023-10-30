package pgdb

import (
	"context"
	"database/sql"
	"errors"
	"questspace/pkg/storage"

	"golang.org/x/xerrors"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx"
)

type Client struct {
	conn *pgx.Conn
}

func NewClient(conn *pgx.Conn) *Client {
	return &Client{conn: conn}
}

func (c *Client) CreateUser(ctx context.Context, req *storage.CreateUserRequest) (*storage.User, error) {
	_, err := c.GetUser(ctx, &storage.GetUserRequest{Username: req.Username})
	if err == nil {
		return nil, storage.ErrExists
	}
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return nil, xerrors.Errorf("failed to get user: %w", err)
	}
	if err := c.conn.WaitUntilReady(ctx); err != nil {
		return nil, xerrors.Errorf("failed to await db readiness: %w", err)
	}

	values := []interface{}{req.Username, []byte(req.Password)}
	query := sq.
		Insert(`"user"`).
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
	row, err := c.conn.QueryEx(ctx, queryStr, nil, args...)
	if err != nil {
		return nil, xerrors.Errorf("failed to execute query %s: %w", queryStr, err)
	}
	defer row.Close()
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
	if err := c.conn.WaitUntilReady(ctx); err != nil {
		return nil, xerrors.Errorf("failed to await db readiness: %w", err)
	}
	query := sq.
		Select("id", "username", "avatar_url").
		From(`"user"`).
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
	row, err := c.conn.QueryEx(ctx, queryStr, nil, args...)
	if err != nil {
		return nil, xerrors.Errorf("failed to execute query %s: %w", queryStr, err)
	}
	defer row.Close()
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
	if err := c.conn.WaitUntilReady(ctx); err != nil {
		return nil, xerrors.Errorf("failed to await db readiness: %w", err)
	}
	pwQuery := sq.
		Select("password").
		From(`"user"`).
		Where(sq.Eq{"id": req.Id}).
		PlaceholderFormat(sq.Dollar)
	queryStr, args, err := pwQuery.ToSql()
	if err != nil {
		return nil, xerrors.Errorf("failed to build query string: %w", err)
	}
	row := c.conn.QueryRowEx(ctx, queryStr, nil, args...)
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
		Update(`"user"`).
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
	rows, err := c.conn.QueryEx(ctx, queryStr, nil, args...)
	if err != nil {
		return nil, xerrors.Errorf("failed to execute query %s: %w", queryStr, err)
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, xerrors.Errorf("failed to insert row: %w", rows.Err())
	}

	user := &storage.User{}
	if err := rows.Scan(&user.Id, &user.Username, &user.AvatarURL); err != nil {
		return nil, xerrors.Errorf("failed to scan row: %w", err)
	}
	return user, nil
}
