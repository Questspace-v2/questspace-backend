package pgclient

import (
	"context"
	"database/sql"
	"errors"

	sq "github.com/Masterminds/squirrel"
	"golang.org/x/xerrors"

	"questspace/pkg/storage"
)

func (c *Client) CreateQuest(ctx context.Context, req *storage.CreateQuestRequest) (*storage.Quest, error) {
	values := []any{
		req.Name,
		req.Description,
		req.MediaLink,
		req.RegistrationDeadline,
		req.StartTime,
		req.FinishTime,
		string(req.Access),
		req.Creator.ID,
		req.HasBrief,
		req.Brief,
		req.QuestType,
	}
	query := sq.Insert("questspace.quest").
		Columns(
			"name",
			"description",
			"media_link",
			"registration_deadline",
			"start_time",
			"finish_time",
			"access",
			"creator_id",
			"has_brief",
			"brief",
			"quest_type",
		).
		Suffix("RETURNING id").
		PlaceholderFormat(sq.Dollar)
	if req.MaxTeamCap != nil {
		values = append(values, *req.MaxTeamCap)
		query = query.Columns("max_team_cap")
	}
	if req.MaxTeamsAmount != nil && *req.MaxTeamsAmount > 0 {
		values = append(values, *req.MaxTeamsAmount)
		query = query.Columns("max_teams_amount")
	}
	if req.RegistrationType != storage.RegistrationUnspecified {
		values = append(values, req.RegistrationType)
		query = query.Columns("registration_type")
	}

	row := query.Values(values...).RunWith(c.runner).QueryRowContext(ctx)
	quest := storage.Quest{
		Name:        req.Name,
		Description: req.Description,
		MediaLink:   req.MediaLink,
		StartTime:   req.StartTime,
		FinishTime:  req.FinishTime,
		Access:      req.Access,
		Creator: &storage.User{
			ID:        req.Creator.ID,
			Username:  req.Creator.Username,
			AvatarURL: req.Creator.AvatarURL,
		},
		RegistrationDeadline: req.RegistrationDeadline,
		MaxTeamCap:           req.MaxTeamCap,
		HasBrief:             req.HasBrief,
		Brief:                req.Brief,
		MaxTeamsAmount:       req.MaxTeamsAmount,
		RegistrationType:     req.RegistrationType,
		QuestType:            req.QuestType,
	}
	if err := row.Scan(&quest.ID); err != nil {
		return nil, xerrors.Errorf("scan row: %w", err)
	}
	if quest.RegistrationType == storage.RegistrationUnspecified {
		quest.RegistrationType = storage.RegistrationAuto
	}
	return &quest, nil
}

const getQuestQuery = `
SELECT 
	q.id, 
	q.name, 
	q.description, 
	q.media_link, 
	q.registration_deadline, 
	q.start_time, 
	q.finish_time,
	q.access,
	q.max_team_cap,
	q.finished,
	q.has_brief,
	q.brief,
	q.max_teams_amount,
	q.registration_type,
	q.quest_type,
	u.id,
	u.username,
	u.avatar_url
FROM questspace.quest q LEFT JOIN questspace.user u ON u.id = q.creator_id
	WHERE q.id = $1
`

func (c *Client) GetQuest(ctx context.Context, req *storage.GetQuestRequest) (*storage.Quest, error) {
	row := c.runner.QueryRowContext(ctx, getQuestQuery, req.ID)
	var (
		q                     storage.Quest
		creatorName           sql.NullString
		userId, userAvatarURL sql.NullString
		finished              bool
	)
	if err := row.Scan(
		&q.ID,
		&q.Name,
		&q.Description,
		&q.MediaLink,
		&q.RegistrationDeadline,
		&q.StartTime,
		&q.FinishTime,
		&q.Access,
		&q.MaxTeamCap,
		&finished,
		&q.HasBrief,
		&q.Brief,
		&q.MaxTeamsAmount,
		&q.RegistrationType,
		&q.QuestType,
		&userId,
		&creatorName,
		&userAvatarURL,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, xerrors.Errorf("scan row: %w", err)
	}
	if userId.Valid {
		q.Creator = &storage.User{ID: storage.ID(userId.String)}
	}
	if q.Creator != nil && userAvatarURL.Valid {
		q.Creator.AvatarURL = userAvatarURL.String
	}
	if q.Creator != nil && userAvatarURL.Valid {
		q.Creator.Username = creatorName.String
	}
	if finished {
		q.Status = storage.StatusFinished
	}
	return &q, nil
}

func (c *Client) addAllQuestsCond(query sq.SelectBuilder, userID storage.ID) sq.SelectBuilder {
	const allExpr = `(q.access = 'public' OR (q.access = 'link_only' AND EXISTS(
	SELECT 1 FROM questspace.registration r
		LEFT JOIN questspace.team t ON t.id = r.team_id
		WHERE t.quest_id = q.id AND r.user_id = ?
)))`
	query = query.Where(sq.Expr(allExpr, userID))
	return query
}

func (c *Client) addRegisteredQuestsCond(query sq.SelectBuilder, userID storage.ID) sq.SelectBuilder {
	const registeredExpr = `EXISTS(
	SELECT 1 FROM questspace.registration r
		LEFT JOIN questspace.team t ON t.id = r.team_id
		WHERE t.quest_id = q.id AND r.user_id = ?
)`
	query = query.Where(sq.Expr(registeredExpr, userID))
	return query
}

func (c *Client) addOwnedQuestsCond(query sq.SelectBuilder, userID storage.ID) sq.SelectBuilder {
	query = query.Where(sq.Eq{"q.creator_id": userID})
	return query
}

func (c *Client) GetQuests(ctx context.Context, req *storage.GetQuestsRequest) (*storage.GetQuestsResponse, error) {
	query := sq.Select(
		"q.id",
		"q.name",
		"q.description",
		"q.access",
		"q.registration_deadline",
		"q.start_time",
		"q.finish_time",
		"q.media_link",
		"q.max_team_cap",
		"q.finished",
		"q.quest_type",
		"q.creator_id",
		"u.username",
		"u.avatar_url",
	).From("questspace.quest q").LeftJoin("questspace.user u ON q.creator_id = u.id").
		OrderBy("q.finished", "q.start_time").
		Limit(uint64(req.PageSize)).
		PlaceholderFormat(sq.Dollar)
	if req.Page != nil {
		query = query.Where(sq.And{
			sq.GtOrEq{"q.finished": req.Page.Finished},
			sq.Expr(`q.start_time > to_timestamp(?)`, req.Page.Timestamp),
		})
	}
	switch req.Type {
	case storage.GetPublic:
		query = query.Where(sq.Eq{"q.access": "public"})
	case storage.GetAll:
		query = c.addAllQuestsCond(query, req.User.ID)
	case storage.GetRegistered:
		query = c.addRegisteredQuestsCond(query, req.User.ID)
	case storage.GetOwned:
		query = c.addOwnedQuestsCond(query, req.User.ID)
	}

	rows, err := query.RunWith(c.runner).QueryContext(ctx)
	if err != nil {
		return nil, xerrors.Errorf("query rows: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var quests []storage.Quest
	var (
		username              sql.NullString
		userId, userAvatarURL sql.NullString
		finished              bool
	)

	for rows.Next() {
		var q storage.Quest

		if err := rows.Scan(
			&q.ID,
			&q.Name,
			&q.Description,
			&q.Access,
			&q.RegistrationDeadline,
			&q.StartTime,
			&q.FinishTime,
			&q.MediaLink,
			&q.MaxTeamCap,
			&finished,
			&q.QuestType,
			&userId, &username, &userAvatarURL,
		); err != nil {
			return nil, xerrors.Errorf("scan row: %w", err)
		}

		if userId.Valid {
			q.Creator = &storage.User{ID: storage.ID(userId.String)}
		}
		if q.Creator != nil && username.Valid {
			q.Creator.Username = username.String
		}
		if q.Creator != nil && userAvatarURL.Valid {
			q.Creator.AvatarURL = userAvatarURL.String
		}

		if finished {
			q.Status = storage.StatusFinished
		}
		quests = append(quests, q)
	}
	if err := rows.Err(); err != nil {
		return nil, xerrors.Errorf("iter rows: %w", err)
	}

	var page *storage.Page
	if len(quests) > 0 {
		lastQuest := quests[len(quests)-1]
		page = &storage.Page{
			Finished:  lastQuest.Status == storage.StatusFinished,
			Timestamp: lastQuest.StartTime.UTC().Unix(),
		}
	}

	return &storage.GetQuestsResponse{Quests: quests, NextPage: page}, nil
}

func (c *Client) UpdateQuest(ctx context.Context, req *storage.UpdateQuestRequest) (*storage.Quest, error) {
	query := sq.Update("questspace.quest").
		Where(sq.Eq{"id": req.ID}).
		Suffix(`RETURNING
		id,
		name,
		description,
		media_link,
		creator_id,
		registration_deadline,
		start_time,
		finish_time,
		access,
		max_team_cap,
		finished,
		has_brief,
		brief,
		max_teams_amount,
		registration_type,
		quest_type`).
		PlaceholderFormat(sq.Dollar)
	if len(req.Name) > 0 {
		query = query.Set("name", req.Name)
	}
	if len(req.MediaLink) > 0 {
		query = query.Set("media_link", req.MediaLink)
	}
	if len(req.Description) > 0 {
		query = query.Set("description", req.Description)
	}
	if len(req.Access) > 0 {
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
	if req.HasBrief != nil {
		query = query.Set("has_brief", *req.HasBrief)
	}
	if req.Brief != nil {
		query = query.Set("brief", *req.Brief)
	}
	if req.MaxTeamsAmount != nil && *req.MaxTeamsAmount > 0 {
		query = query.Set("max_teams_amount", *req.MaxTeamsAmount)
	} else if req.MaxTeamsAmount != nil {
		query = query.Set("max_teams_amount", nil)
	}
	if req.RegistrationType != storage.RegistrationUnspecified {
		query = query.Set("registration_type", req.RegistrationType)
	}
	if len(req.QuestType) > 0 {
		query = query.Set("quest_type", req.QuestType)
	}

	row := query.RunWith(c.runner).QueryRowContext(ctx)
	var (
		q         storage.Quest
		creatorID sql.NullString
		finished  bool
	)
	if err := row.Scan(
		&q.ID,
		&q.Name,
		&q.Description,
		&q.MediaLink,
		&creatorID,
		&q.RegistrationDeadline,
		&q.StartTime,
		&q.FinishTime,
		&q.Access,
		&q.MaxTeamCap,
		&finished,
		&q.HasBrief,
		&q.Brief,
		&q.MaxTeamsAmount,
		&q.RegistrationType,
		&q.QuestType,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, xerrors.Errorf("scan row: %w", err)
	}
	if creatorID.Valid {
		q.Creator = &storage.User{ID: storage.ID(creatorID.String)}
	}
	if finished {
		q.Status = storage.StatusFinished
	}

	return &q, nil
}

func (c *Client) DeleteQuest(ctx context.Context, req *storage.DeleteQuestRequest) error {
	query := `
	DELETE FROM questspace.quest
		WHERE id = $1
`

	if _, err := c.runner.ExecContext(ctx, query, req.ID); err != nil {
		return xerrors.Errorf("delete quest: %w", err)
	}
	return nil
}

func (c *Client) FinishQuest(ctx context.Context, req *storage.FinishQuestRequest) error {
	query := `
	UPDATE questspace.quest SET finished = true
		WHERE id = $1
`

	if _, err := c.runner.ExecContext(ctx, query, req.ID); err != nil {
		return xerrors.Errorf("exec query: %w", err)
	}
	return nil
}
