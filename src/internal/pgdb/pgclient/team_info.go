package pgclient

import (
	"context"
	"database/sql"
	"time"

	"questspace/pkg/storage"

	sq "github.com/Masterminds/squirrel"
	"golang.org/x/xerrors"
)

func (c *Client) UpsertTeamInfo(ctx context.Context, req *storage.UpsertTeamInfoRequest) (*storage.TaskGroupTeamInfo, error) {
	query := `
	INSERT INTO questspace.task_group_team_info (team_id, group_id, opening_time, closing_time)
	VALUES ($1, $2, $3, $4) 
	ON CONFLICT (team_id, group_id) DO UPDATE SET opening_time = $3, closing_time = $4
	`

	_, err := c.runner.ExecContext(ctx, query, req.TeamID, req.TaskGroupID, req.OpeningTime, req.ClosingTime)
	if err != nil {
		return nil, err
	}

	resp := &storage.TaskGroupTeamInfo{
		OpeningTime: req.OpeningTime,
		ClosingTime: req.ClosingTime,
	}
	return resp, nil
}

func (c *Client) GetTeamInfo(ctx context.Context, req *storage.GetTeamInfoRequest) (*storage.TaskGroupTeamInfo, error) {
	teamInfos, err := c.GetTeamInfos(ctx, &storage.GetTeamInfosRequest{
		TaskGroupIDs: []storage.ID{req.TaskGroupID},
		TeamData:     req.TeamData,
	})
	if err != nil {
		return nil, xerrors.Errorf("get team infos: %w", err)
	}

	teamInfo, ok := teamInfos[req.TaskGroupID]
	if !ok {
		return nil, storage.ErrNotFound
	}

	return teamInfo, nil
}

func (c *Client) GetTeamInfos(ctx context.Context, req *storage.GetTeamInfosRequest) (storage.GetTeamInfosResponse, error) {
	taskGroups, err := c.GetTaskGroups(ctx, &storage.GetTaskGroupsRequest{QuestID: req.QuestID, GroupIDs: req.TaskGroupIDs})
	if err != nil {
		return nil, xerrors.Errorf("get task groups: %w", err)
	}

	teamInfos, err := c.fillTeamInfos(ctx, taskGroups, req.TeamData)
	if err != nil {
		return nil, xerrors.Errorf("fill team infos: %w", err)
	}
	return teamInfos, nil
}

func (c *Client) fillTeamInfos(ctx context.Context, taskGroups []storage.TaskGroup, teamData storage.TeamData) (storage.GetTeamInfosResponse, error) {
	if len(taskGroups) == 0 {
		return nil, nil
	}

	tgIDs := make([]storage.ID, 0, len(taskGroups))
	for _, tg := range taskGroups {
		tgIDs = append(tgIDs, tg.ID)
	}

	query := sq.Select("tg.id", "ti.opening_time", "ti.closing_time").
		From("questspace.task_group_team_info ti").
		LeftJoin("questspace.task_group tg ON ti.group_id = tg.id").
		Where(sq.Eq{"tg.id": tgIDs}).
		OrderBy("tg.order_idx ASC").
		PlaceholderFormat(sq.Dollar)

	switch {
	case teamData.TeamID != nil:
		query = query.Where(sq.Eq{"ti.team_id": *teamData.TeamID})
	case teamData.UserID != nil:
		query = query.LeftJoin("questspace.team tm ON ti.team_id = tm.id").
			LeftJoin("questspace.registration r ON r.team_id = tm.id").
			Where(sq.Eq{"r.user_id": *teamData.UserID})
	}

	rows, err := query.RunWith(c.runner).QueryContext(ctx)
	if err != nil {
		return nil, xerrors.Errorf("query rows: %w", err)
	}
	defer func() { _ = rows.Close() }()

	groupToTeams := make(map[storage.ID]*storage.TaskGroupTeamInfo, len(taskGroups))
	var openingTime, closingTime sql.NullTime
	var tgID storage.ID
	for rows.Next() {
		var teamInfo *storage.TaskGroupTeamInfo
		if err = rows.Scan(
			&tgID,
			&openingTime,
			&closingTime,
		); err != nil {
			return nil, xerrors.Errorf("scan rows: %w", err)
		}

		if openingTime.Valid {
			teamInfo = &storage.TaskGroupTeamInfo{
				OpeningTime: openingTime.Time,
			}
		}
		if closingTime.Valid {
			teamInfo.ClosingTime = &closingTime.Time
		}

		groupToTeams[tgID] = teamInfo
	}
	if err = rows.Err(); err != nil {
		return nil, xerrors.Errorf("iter rows: %w", err)
	}

	row := c.runner.QueryRowContext(ctx,
		"SELECT q.start_time FROM questspace.task_group tg LEFT JOIN questspace.quest q ON q.id = tg.quest_id WHERE tg.id = $1",
		tgIDs[0],
	)
	var startTime time.Time
	if err = row.Scan(&startTime); err != nil {
		return nil, xerrors.Errorf("scan quest row: %w", err)
	}

	var prevClosingTime *time.Time
	var alreadySetStartTime bool
	first := true
	groupToTeamInfo := make(map[storage.ID]*storage.TaskGroupTeamInfo, len(taskGroups))
	for _, tg := range taskGroups {
		if tg.Sticky {
			continue
		}

		ti, ok := groupToTeams[tg.ID]
		if !ok && prevClosingTime != nil {
			ti = &storage.TaskGroupTeamInfo{
				OpeningTime: *prevClosingTime,
			}
			prevClosingTime = nil
		} else if !ok && !alreadySetStartTime && first {
			ti = &storage.TaskGroupTeamInfo{
				OpeningTime: startTime,
			}
			alreadySetStartTime = true
		}

		first = false
		if ti == nil {
			continue
		}

		if ti.ClosingTime != nil {
			prevClosingTime = ti.ClosingTime
		}

		groupToTeamInfo[tg.ID] = ti
	}

	return groupToTeamInfo, nil
}
