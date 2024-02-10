package pgdb

import (
	"context"
	"database/sql"
	"errors"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx"
	"github.com/spkg/ptr"
	"golang.org/x/xerrors"

	"questspace/pkg/storage"
)

const uniqueViolationCode = "23505"

type Client struct {
	runner sq.RunnerContext
}

var _ storage.QuestSpaceStorage = &Client{}

func NewClient(r sq.RunnerContext) *Client {
	return &Client{runner: r}
}

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
		pgErr := &pgx.PgError{}
		if errors.As(err, pgErr) && pgErr.Code == uniqueViolationCode {
			return nil, storage.ErrExists
		}
		return nil, xerrors.Errorf("failed to scan row: %w", err)
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
		return nil, xerrors.New("at least one of request fields must not be empty")
	}
	row := query.RunWith(c.runner).QueryRowContext(ctx)

	user := storage.User{}
	if err := row.Scan(&user.ID, &user.Username, &user.AvatarURL); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, xerrors.Errorf("failed to scan row: %w", err)
	}
	return &user, nil
}

func (c *Client) UpdateUser(ctx context.Context, req *storage.UpdateUserRequest) (*storage.User, error) {
	if req.Username == "" && req.Password == "" && req.AvatarURL == "" {
		return nil, xerrors.New("nothing to change!")
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
		return nil, xerrors.Errorf("failed to scan row: %w", err)
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
		return "", xerrors.New("either user id or username should be present")
	}
	row := query.RunWith(c.runner).QueryRowContext(ctx)

	var pw []byte
	if err := row.Scan(&pw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", storage.ErrNotFound
		}
		return "", xerrors.Errorf("failed to scan row: %w", err)
	}
	return string(pw), nil
}

func (c *Client) DeleteUser(ctx context.Context, req *storage.DeleteUserRequest) error {
	query := sq.Delete("questspace.user").
		Where(sq.Eq{"id": req.ID}).
		PlaceholderFormat(sq.Dollar)

	if _, err := query.RunWith(c.runner).ExecContext(ctx); err != nil {
		return xerrors.Errorf("failed to delete user: %w", err)
	}
	return nil
}

func (c *Client) CreateQuest(ctx context.Context, req *storage.CreateQuestRequest) (*storage.Quest, error) {
	values := []interface{}{req.Name, req.Description, req.MediaLink, req.RegistrationDeadline, req.StartTime, req.FinishTime, req.Access, req.Creator.Username}
	query := sq.Insert("questspace.quest").
		Columns("name", "description", "media_link", "registration_deadline", "start_time", "finish_time", "access", "creator").
		Suffix("RETURNING id").
		PlaceholderFormat(sq.Dollar)
	if req.MaxTeamCap != nil {
		values = append(values, *req.MaxTeamCap)
		query = query.Columns("max_team_cap")
	}

	row := query.Values(values...).RunWith(c.runner).QueryRowContext(ctx)
	quest := storage.Quest{
		Name:                 req.Name,
		Description:          req.Description,
		MediaLink:            req.MediaLink,
		StartTime:            req.StartTime,
		FinishTime:           req.FinishTime,
		Access:               req.Access,
		Creator:              &storage.User{Username: req.Creator.Username},
		RegistrationDeadline: req.RegistrationDeadline,
		MaxTeamCap:           req.MaxTeamCap,
	}
	if err := row.Scan(&quest.ID); err != nil {
		return nil, xerrors.Errorf("failed to scan row: %w", err)
	}
	return &quest, nil
}

func (c *Client) GetQuest(ctx context.Context, req *storage.GetQuestRequest) (*storage.Quest, error) {
	query := sq.Select(
		"q.id", "q.name", "q.description", "q.media_link", "q.registration_deadline",
		"q.start_time", "q.finish_time", "q.access", "q.max_team_cap",
		"u.id", "u.avatar_url",
	).From("questspace.quest q").
		LeftJoin("questspace.user u ON u.username = q.creator").
		Where(sq.Eq{"q.id": req.ID}).
		PlaceholderFormat(sq.Dollar)

	row := query.RunWith(c.runner).QueryRowContext(ctx)
	var (
		q                     storage.Quest
		userId, userAvatarURL sql.NullString
		regDeadline, finTime  sql.NullTime
		maxTeamCap            sql.NullInt32
	)
	if err := row.Scan(&q.ID, &q.Name, &q.Description, &q.MediaLink, &regDeadline,
		&q.StartTime, &finTime, &q.Access, &maxTeamCap, &userId, &userAvatarURL); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, xerrors.Errorf("failed to scan row: %w", err)
	}
	if userId.Valid {
		q.Creator = &storage.User{ID: userId.String}
	}
	if q.Creator != nil && userAvatarURL.Valid {
		q.Creator.AvatarURL = userAvatarURL.String
	}
	if regDeadline.Valid {
		q.RegistrationDeadline = &regDeadline.Time
	}
	if finTime.Valid {
		q.FinishTime = &finTime.Time
	}
	if maxTeamCap.Valid {
		q.MaxTeamCap = ptr.Int(int(maxTeamCap.Int32))
	}

	return &q, nil
}

func (c *Client) UpdateQuest(ctx context.Context, req *storage.UpdateQuestRequest) (*storage.Quest, error) {
	query := sq.Update("questspace.quest").
		Where(sq.Eq{"id": req.ID}).
		Suffix("RETURNING id, name, description, media_link, creator_name, " +
			"registration_deadline, start_time, finish_time, access, max_team_cap").
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

	row := query.RunWith(c.runner).QueryRowContext(ctx)
	var (
		q                    storage.Quest
		creatorName          sql.NullString
		regDeadline, finTime sql.NullTime
		maxTeamCap           sql.NullInt32
	)
	if err := row.Scan(&q.ID, &q.Name, &q.Description, &q.MediaLink, &creatorName,
		&regDeadline, &q.StartTime, &finTime, &q.Access, &maxTeamCap); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, xerrors.Errorf("failed to scan row: %w", err)
	}
	if regDeadline.Valid {
		q.RegistrationDeadline = &regDeadline.Time
	}
	if finTime.Valid {
		q.FinishTime = &finTime.Time
	}
	if maxTeamCap.Valid {
		q.MaxTeamCap = ptr.Int(int(maxTeamCap.Int32))
	}
	if creatorName.Valid {
		q.Creator = &storage.User{Username: creatorName.String}
	}

	return &q, nil
}

func (c *Client) DeleteQuest(ctx context.Context, req *storage.DeleteQuestRequest) error {
	query := sq.Delete("questspace.quest").
		Where(sq.Eq{"id": req.ID}).
		PlaceholderFormat(sq.Dollar)

	if _, err := query.RunWith(c.runner).ExecContext(ctx); err != nil {
		return xerrors.Errorf("failed to delete quest: %w", err)
	}
	return nil
}
