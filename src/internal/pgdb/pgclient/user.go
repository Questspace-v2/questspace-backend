package pgclient

import (
	"context"
	"database/sql"
	"errors"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/xerrors"

	"questspace/pkg/storage"
)

func (c *Client) CreateUser(ctx context.Context, req *storage.CreateUserRequest) (*storage.User, error) {
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
	row := query.Values(values...).RunWith(c.runner).QueryRowContext(ctx)

	var id string
	if err := row.Scan(&id); err != nil {
		if pgErr := new(pgconn.PgError); errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
			return nil, storage.ErrExists
		}
		return nil, xerrors.Errorf("scan row: %w", err)
	}
	user := storage.User{
		ID:        id,
		Username:  req.Username,
		AvatarURL: req.AvatarURL,
	}
	return &user, nil
}

func (c *Client) GetUser(ctx context.Context, req *storage.GetUserRequest) (*storage.User, error) {
	query := sq.
		Select("id", "username", "avatar_url").
		From("questspace.user").
		PlaceholderFormat(sq.Dollar)
	if req.ID != "" {
		query = query.Where(sq.Eq{"id": req.ID})
	} else if req.Username != "" {
		query = query.Where(sq.Eq{"username": req.Username})
	} else {
		return nil, xerrors.Errorf("at least one of request fields must not be empty: %w", storage.ErrValidation)
	}
	row := query.RunWith(c.runner).QueryRowContext(ctx)

	user := storage.User{}
	if err := row.Scan(&user.ID, &user.Username, &user.AvatarURL); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, xerrors.Errorf("scan row: %w", err)
	}
	return &user, nil
}

func (c *Client) UpdateUser(ctx context.Context, req *storage.UpdateUserRequest) (*storage.User, error) {
	if req.Username == "" && req.Password == "" && req.AvatarURL == "" {
		return nil, xerrors.Errorf("nothing to change: %w", storage.ErrValidation)
	}

	query := sq.
		Update("questspace.user").
		Where(sq.Eq{"id": req.ID}).
		Suffix("RETURNING id, username, avatar_url").
		PlaceholderFormat(sq.Dollar)
	if req.Username != "" {
		query = query.Set("username", req.Username)
	}
	if req.Password != "" {
		query = query.Set("password", []byte(req.Password))
	}
	if req.AvatarURL != "" {
		query = query.Set("avatar_url", req.AvatarURL)
	}
	row := query.RunWith(c.runner).QueryRowContext(ctx)

	user := storage.User{}
	if err := row.Scan(&user.ID, &user.Username, &user.AvatarURL); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		if pgErr := new(pgconn.PgError); errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
			return nil, storage.ErrExists
		}
		return nil, xerrors.Errorf("scan row: %w", err)
	}
	return &user, nil
}

func (c *Client) GetUserPasswordHash(ctx context.Context, req *storage.GetUserRequest) (string, error) {
	query := sq.Select("password").
		From("questspace.user").
		PlaceholderFormat(sq.Dollar)
	if req.ID != "" {
		query = query.Where(sq.Eq{"id": req.ID})
	} else if req.Username != "" {
		query = query.Where(sq.Eq{"username": req.Username})
	} else {
		return "", xerrors.Errorf("either user id or username should be present: %w", storage.ErrValidation)
	}
	row := query.RunWith(c.runner).QueryRowContext(ctx)

	var pw []byte
	if err := row.Scan(&pw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", storage.ErrNotFound
		}
		return "", xerrors.Errorf("scan row: %w", err)
	}
	return string(pw), nil
}

func (c *Client) CreateOrUpdateByExternalID(ctx context.Context, req *storage.CreateOrUpdateRequest) (*storage.User, error) {
	sqlQuery := `INSERT INTO questspace.user (username, avatar_url, password, external_id)
	VALUES ($1, $2, $3, $4)
	ON CONFLICT (external_id) DO UPDATE SET password = $3
	RETURNING id, username, avatar_url
`
	expr := sq.Expr(sqlQuery, req.Username, req.AvatarURL, []byte(req.ExternalID), req.ExternalID)

	row := sq.QueryRowContextWith(ctx, c.runner, expr)
	user := storage.User{}
	if err := row.Scan(&user.ID, &user.Username, &user.AvatarURL); err != nil {
		return nil, xerrors.Errorf("scan row: %w", err)
	}

	return &user, nil
}

func (c *Client) DeleteUser(ctx context.Context, req *storage.DeleteUserRequest) error {
	query := sq.Delete("questspace.user").
		Where(sq.Eq{"id": req.ID}).
		PlaceholderFormat(sq.Dollar)

	if _, err := query.RunWith(c.runner).ExecContext(ctx); err != nil {
		return xerrors.Errorf("delete user: %w", err)
	}
	return nil
}
