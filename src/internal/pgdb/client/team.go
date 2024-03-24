package pgdb

import (
	"context"
	"database/sql"
	"errors"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/spkg/ptr"
	"golang.org/x/xerrors"

	"questspace/pkg/storage"
)

const createTeamQuery = `
WITH created_team AS (
	INSERT INTO questspace.team (name, quest_id, cap_id) VALUES ($1, $2, $3)
	RETURNING id, name, link_id, cap_id
) SELECT t.id, t.name, t.link_id, u.id, u.username, u.avatar_url
FROM created_team t LEFT JOIN questspace.user u ON t.cap_id = u.id
`

func (c *Client) CreateTeam(ctx context.Context, req *storage.CreateTeamRequest) (*storage.Team, error) {
	sqlQuery := sq.Expr(createTeamQuery, req.Name, req.QuestID, req.Creator.ID)

	row := sq.QueryRowContextWith(ctx, c.runner, sqlQuery)
	team := &storage.Team{Capitan: &storage.User{}}
	if err := row.Scan(&team.ID, &team.Name, &team.InviteLinkID, &team.Capitan.ID, &team.Capitan.Username, &team.Capitan.AvatarURL); err != nil {
		if pgErr := new(pgconn.PgError); errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
			return nil, storage.ErrExists
		}
		return nil, xerrors.Errorf("scan row: %w", err)
	}

	return team, nil
}

func (c *Client) getTeamMembers(ctx context.Context, teamID string) ([]*storage.User, error) {
	query := sq.Select("u.id", "u.username", "u.avatar_url").
		From("questspace.user u").
		LeftJoin("questspace.registration r ON r.user_id = u.id").
		Where(sq.Eq{"r.team_id": teamID}).
		PlaceholderFormat(sq.Dollar)

	rows, err := query.RunWith(c.runner).QueryContext(ctx)
	if err != nil {
		return nil, xerrors.Errorf("query rows: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var members []*storage.User
	for rows.Next() {
		var member storage.User
		if err := rows.Scan(&member.ID, &member.Username, &member.AvatarURL); err != nil {
			return nil, xerrors.Errorf("scan row: %w", err)
		}
		members = append(members, &member)
	}
	if err := rows.Err(); err != nil {
		return nil, xerrors.Errorf("iter rows: %w", err)
	}
	return members, nil
}

func (c *Client) GetTeam(ctx context.Context, req *storage.GetTeamRequest) (*storage.Team, error) {
	query := sq.Select("t.id", "t.name", "t.invite_path", "t.score", "q.max_team_cap", "u.id", "u.username", "u.avatar_url").
		From("questspace.team t").
		LeftJoin("questspace.quest q ON q.id = t.quest_id").LeftJoin("questspace.user u ON t.cap_id = u.id").
		PlaceholderFormat(sq.Dollar)
	if req.ID != "" {
		query = query.Where(sq.Eq{"t.id": req.ID})
	} else if req.InvitePath != "" {
		query = query.Where(sq.Eq{"t.invite_path": req.InvitePath})
	} else {
		return nil, xerrors.Errorf("no search key was provided: %w", storage.ErrValidation)
	}

	row := query.RunWith(c.runner).QueryRowContext(ctx)
	team := &storage.Team{Quest: &storage.Quest{}, Capitan: &storage.User{}}
	var maxTeamCap sql.NullInt32
	if err := row.Scan(&team.ID, &team.Name, &team.InviteLink, &team.Score, &maxTeamCap, &team.Capitan.ID, &team.Capitan.Username, &team.Capitan.AvatarURL); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, xerrors.Errorf("scan row: %w", err)
	}
	if maxTeamCap.Valid {
		team.Quest.MaxTeamCap = ptr.Int(int(maxTeamCap.Int32))
	}
	if req.IncludeMembers {
		var err error
		team.Members, err = c.getTeamMembers(ctx, team.ID)
		if err != nil {
			return nil, xerrors.Errorf("get team members: %w", err)
		}
	}
	return team, nil
}

func (c *Client) GetTeams(ctx context.Context, req *storage.GetTeamsRequest) ([]*storage.Team, error) {
	query := sq.Select("t.id", "t.name").
		From("questspace.team t").
		LeftJoin("questspace.registration r ON r.team_id = t.id").
		Where(sq.Eq{"r.user_id": req.User.ID}).
		PlaceholderFormat(sq.Dollar)
	if len(req.QuestIDs) > 0 {
		query = query.Where(sq.Eq{"t.quest_id": req.QuestIDs})
	}

	rows, err := query.RunWith(c.runner).QueryContext(ctx)
	if err != nil {
		return nil, xerrors.Errorf("query rows: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var (
		teams []*storage.Team
	)
	for rows.Next() {
		var team storage.Team
		if err := rows.Scan(&team.ID, &team.Name); err != nil {
			return nil, xerrors.Errorf("scan row: %w", err)
		}
		teams = append(teams, &team)
	}
	if err := rows.Err(); err != nil {
		return nil, xerrors.Errorf("iter rows: %w", err)
	}
	return teams, nil
}

func (c *Client) SetInviteLink(ctx context.Context, req *storage.SetInviteLinkRequest) error {
	query := sq.Update("questspace.team").
		Set("invite_path", req.InviteURL).
		Where(sq.Eq{"id": req.TeamID}).
		PlaceholderFormat(sq.Dollar)

	_, err := query.RunWith(c.runner).ExecContext(ctx)
	if err != nil {
		return xerrors.Errorf("exec query: %w", err)
	}
	return nil
}

const joinTeamQuery = `
WITH created_registration AS (
	INSERT INTO questspace.registration (user_id, team_id)
	SELECT $1, t.id FROM questspace.team t
	WHERE t.invite_path = $2
	RETURNING user_id
) SELECT id, username, avatar_url
FROM questspace.user
WHERE id = (
	SELECT user_id FROM created_registration
)
`

func (c *Client) JoinTeam(ctx context.Context, req *storage.JoinTeamRequest) (*storage.User, error) {
	sqlQuery := sq.Expr(joinTeamQuery, req.User.ID, req.InvitePath)

	row := sq.QueryRowContextWith(ctx, c.runner, sqlQuery)
	user := &storage.User{}
	if err := row.Scan(&user.ID, &user.Username, &user.AvatarURL); err != nil {
		if pgErr := new(pgconn.PgError); errors.As(err, &pgErr) && pgErr.Code == triggerActionExceptionCode {
			return nil, storage.ErrTeamAlreadyFull
		}
		return nil, xerrors.Errorf("scan row: %w", err)
	}
	return user, nil
}
