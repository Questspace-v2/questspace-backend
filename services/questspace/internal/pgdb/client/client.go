package pgdb

import (
	"context"
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
	values := []interface{}{req.Username, req.Password}
	query := sq.
		Insert("\"user\"").
		Columns("username", "password").
		Suffix("RETURNING id").
		PlaceholderFormat(sq.Dollar)
	if req.FirstName != "" && req.LastName != "" {
		query = query.Columns("first_name", "last_name")
		values = append(values, req.FirstName, req.LastName)
	}
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
		return nil, storage.ErrNotFound
	}
	var id string
	if err := row.Scan(&id); err != nil {
		return nil, xerrors.Errorf("failed to scan row: %w", err)
	}
	user := storage.User(*req)
	user.Id = id
	return &user, nil
}

func (c *Client) GetUser(ctx context.Context, req *storage.GetUserRequest) (*storage.User, error) {
	if err := c.conn.WaitUntilReady(ctx); err != nil {
		return nil, xerrors.Errorf("failed to await db readiness: %w", err)
	}
	query := sq.
		Select("id", "username", "avatar_url").
		From("\"user\"").
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
	panic("implement me")
}
