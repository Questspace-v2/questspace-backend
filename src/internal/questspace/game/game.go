package game

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/yandex/perforator/library/go/core/xerrors"
	"go.uber.org/zap"

	"questspace/internal/qtime"
	"questspace/pkg/httperrors"
	"questspace/pkg/logging"
	"questspace/pkg/storage"
)

type Service struct {
	ts  storage.TaskStorage
	tgs storage.TaskGroupStorage
	tms storage.TeamStorage
	ah  storage.AnswerHintStorage
}

func NewService(ts storage.TaskStorage, tgs storage.TaskGroupStorage, tms storage.TeamStorage, ah storage.AnswerHintStorage) *Service {
	return &Service{
		ts:  ts,
		tgs: tgs,
		tms: tms,
		ah:  ah,
	}
}

type AnswerDataRequest struct {
	Quest      *storage.Quest
	Team       *storage.Team
	TaskGroups []storage.TaskGroup
}

type AnswerTaskHint struct {
	Taken   bool                 `json:"taken"`
	Name    string               `json:"name,omitempty"`
	Text    string               `json:"text,omitempty"`
	Penalty storage.PenaltyOneOf `json:"penalty"`
}

type AnswerTask struct {
	ID           storage.ID               `json:"id"`
	OrderIdx     int                      `json:"order_idx"`
	Name         string                   `json:"name"`
	Question     string                   `json:"question"`
	Reward       int                      `json:"reward"`
	Verification storage.VerificationType `json:"verification" enums:"auto,manual"`
	Hints        []AnswerTaskHint         `json:"hints"`
	Accepted     bool                     `json:"accepted"`
	Score        int                      `json:"score"`
	Answer       string                   `json:"answer,omitempty"`
	PubTime      *time.Time               `json:"pub_time,omitempty"`
	MediaLinks   []string                 `json:"media_links,omitempty"`
	// Deprecated
	MediaLink string `json:"media_link,omitempty" example:"deprecated"`
	// Deprecated
	VerificationType storage.VerificationType `json:"verification_type" example:"deprecated"`
}

type AnswerTaskGroup struct {
	ID           storage.ID                 `json:"id"`
	OrderIdx     int                        `json:"order_idx"`
	Name         string                     `json:"name"`
	Description  string                     `json:"description,omitempty"`
	PubTime      *time.Time                 `json:"pub_time,omitempty"`
	Sticky       bool                       `json:"sticky,omitempty"`
	Tasks        []AnswerTask               `json:"tasks"`
	HasTimeLimit bool                       `json:"has_time_limit,omitempty"`
	TimeLimit    *storage.Duration          `json:"time_limit,omitempty" swaggertype:"string" example:"45m"`
	TeamInfo     *storage.TaskGroupTeamInfo `json:"team_info,omitempty"`
}

type AnswerDataResponse struct {
	Quest      *storage.Quest    `json:"quest"`
	Team       *storage.Team     `json:"team"`
	TaskGroups []AnswerTaskGroup `json:"task_groups,omitempty"`
}

func (s *Service) FillAnswerData(ctx context.Context, req *AnswerDataRequest) (*AnswerDataResponse, error) {
	takenHints, err := s.ah.GetHintTakes(ctx, &storage.GetHintTakesRequest{TeamID: req.Team.ID, QuestID: req.Quest.ID})
	if err != nil {
		return nil, xerrors.Errorf("get hints: %w", err)
	}
	acceptedTasks, err := s.ah.GetAcceptedTasks(ctx, &storage.GetAcceptedTasksRequest{TeamID: req.Team.ID, QuestID: req.Quest.ID})
	if err != nil {
		return nil, xerrors.Errorf("get accepted tasks: %w", err)
	}
	return s.fillAnswerData(ctx, req, takenHints, acceptedTasks), nil
}

func (s *Service) fillAnswerData(ctx context.Context, req *AnswerDataRequest, takenHints storage.HintTakes, acceptedTasks storage.AcceptedTasks) *AnswerDataResponse {
	taskGroups := make([]AnswerTaskGroup, 0, len(req.TaskGroups))
	var nextStart *time.Time
	now := qtime.Now()
	for _, tg := range req.TaskGroups {
		newTg := AnswerTaskGroup{
			ID:           tg.ID,
			OrderIdx:     tg.OrderIdx,
			Name:         tg.Name,
			Description:  tg.Description,
			PubTime:      tg.PubTime,
			Sticky:       tg.Sticky,
			Tasks:        make([]AnswerTask, 0, len(tg.Tasks)),
			HasTimeLimit: tg.HasTimeLimit,
			TimeLimit:    tg.TimeLimit,
			TeamInfo:     tg.TeamInfo,
		}

		for _, t := range tg.Tasks {
			newT := AnswerTask{
				ID:               t.ID,
				OrderIdx:         t.OrderIdx,
				Name:             t.Name,
				Question:         t.Question,
				Reward:           t.Reward,
				Verification:     t.Verification,
				VerificationType: t.Verification,
				Hints:            make([]AnswerTaskHint, len(t.FullHints)),
				PubTime:          t.PubTime,
				MediaLink:        t.MediaLink,
				MediaLinks:       t.MediaLinks,
			}
			if ans, ok := acceptedTasks[t.ID]; ok {
				newT.Accepted = true
				newT.Answer = ans.Text
				newT.Score = ans.Score
			}
			for _, h := range takenHints[newT.ID] {
				newT.Hints[h.Hint.Index].Taken = true
				newT.Hints[h.Hint.Index].Text = h.Hint.Text
			}
			for i, hint := range t.FullHints {
				newT.Hints[i].Penalty = hint.Penalty
				if hint.Name != nil {
					newT.Hints[i].Name = *hint.Name
				}
			}

			newTg.Tasks = append(newTg.Tasks, newT)
		}
		if req.Quest.QuestType != storage.TypeLinear || newTg.Sticky {
			taskGroups = append(taskGroups, newTg)
			continue
		}
		if newTg.TeamInfo == nil && nextStart != nil {
			newTg.TeamInfo = &storage.TaskGroupTeamInfo{
				OpeningTime: *nextStart,
			}
			nextStart = nil
		}
		if newTg.TeamInfo != nil && newTg.TeamInfo.ClosingTime != nil {
			taskGroups = append(taskGroups, newTg)
			continue
		}
		if newTg.HasTimeLimit && newTg.TimeLimit != nil {
			deadline := newTg.TeamInfo.OpeningTime.Add(time.Duration(*newTg.TimeLimit))
			if deadline.Before(now) {
				newTg.TeamInfo.ClosingTime = &deadline
				if _, err := s.tgs.UpsertTeamInfo(ctx, &storage.UpsertTeamInfoRequest{
					TeamID:      req.Team.ID,
					TaskGroupID: newTg.ID,
					OpeningTime: newTg.TeamInfo.OpeningTime,
					ClosingTime: &deadline,
				}); err != nil {
					logging.Error(ctx, "error updating closing time", zap.Error(err))
				}
				taskGroups = append(taskGroups, newTg)
				nextStart = &deadline
				continue
			}
		}
		taskGroups = append(taskGroups, newTg)
		break
	}

	resp := &AnswerDataResponse{
		Quest:      req.Quest,
		Team:       req.Team,
		TaskGroups: taskGroups,
	}
	return resp
}

type TaskResult struct {
	Score      int
	groupIndex int
	taskIndex  int
}

type TeamResult struct {
	TeamID                storage.ID
	TeamName              string
	TotalScore            int
	TaskScore             int
	Penalty               int
	TaskResults           []TaskResult
	lastCorrectAnswerTime *time.Time
}

func (t TeamResult) MarshalJSON() ([]byte, error) {
	resJSONMap := make(map[string]interface{}, 5+len(t.TaskResults))
	resJSONMap["team_id"] = t.TeamID
	resJSONMap["team_name"] = t.TeamName
	resJSONMap["total_score"] = t.TotalScore
	resJSONMap["task_score"] = t.TaskScore
	resJSONMap["penalty"] = t.Penalty
	for _, result := range t.TaskResults {
		taskKey := fmt.Sprintf("task_%d_%d_score", result.groupIndex, result.taskIndex)
		resJSONMap[taskKey] = result.Score
	}

	res, err := json.Marshal(resJSONMap)
	if err != nil {
		return nil, err
	}
	return res, nil
}

type TeamResults struct {
	Results    []TeamResult        `json:"results"`
	TaskGroups []storage.TaskGroup `json:"task_groups"`
}

func (s *Service) GetResults(ctx context.Context, questID storage.ID) (*TeamResults, error) {
	teams, err := s.tms.GetTeams(ctx, &storage.GetTeamsRequest{QuestIDs: []storage.ID{questID}, AcceptedOnly: true})
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, httperrors.Errorf(http.StatusNotFound, "quest %q not found", questID)
		}
		return nil, xerrors.Errorf("get teams: %w", err)
	}
	taskGroups, err := s.tgs.GetTaskGroups(ctx, &storage.GetTaskGroupsRequest{QuestID: questID, IncludeTasks: true})
	if err != nil {
		return nil, xerrors.Errorf("get task groups: %w", err)
	}
	results, err := s.ah.GetScoreResults(ctx, &storage.GetResultsRequest{QuestID: questID})
	if err != nil {
		return nil, xerrors.Errorf("get results: %w", err)
	}
	penalties, err := s.ah.GetPenalties(ctx, &storage.GetPenaltiesRequest{QuestID: questID})
	if err != nil {
		return nil, xerrors.Errorf("get penalties: %w", err)
	}

	var res TeamResults
	for _, team := range teams {
		teamScore := results[team.ID]
		teamPenalties := penalties[team.ID]
		teamRes := TeamResult{
			TeamID:   team.ID,
			TeamName: team.Name,
		}
		for i, tg := range taskGroups {
			for j, task := range tg.Tasks {
				taskRes := TaskResult{
					groupIndex: i,
					taskIndex:  j,
				}
				if scoreRes, ok := teamScore[task.ID]; ok {
					taskRes.Score = scoreRes.Score
					teamRes.TaskScore += scoreRes.Score
					teamRes.TotalScore += scoreRes.Score
					if teamRes.lastCorrectAnswerTime == nil {
						teamRes.lastCorrectAnswerTime = scoreRes.ScoreTime
					} else if teamRes.lastCorrectAnswerTime.Before(*scoreRes.ScoreTime) {
						teamRes.lastCorrectAnswerTime = scoreRes.ScoreTime
					}
				}
				teamRes.TaskResults = append(teamRes.TaskResults, taskRes)
			}
		}
		for _, p := range teamPenalties {
			teamRes.Penalty += p.Value
			teamRes.TotalScore -= p.Value
		}
		res.Results = append(res.Results, teamRes)
	}

	sort.Slice(res.Results, func(i, j int) bool {
		resScoreL, resScoreR := res.Results[i].TotalScore, res.Results[j].TotalScore
		if resScoreL != resScoreR {
			return resScoreL >= resScoreR
		}
		if res.Results[i].lastCorrectAnswerTime != nil && res.Results[j].lastCorrectAnswerTime != nil {
			return res.Results[i].lastCorrectAnswerTime.Before(*res.Results[j].lastCorrectAnswerTime)
		}
		return res.Results[i].TeamName >= res.Results[j].TeamName
	})
	res.TaskGroups = taskGroups
	return &res, nil
}

type LeaderboardRow struct {
	TeamID                storage.ID `json:"team_id"`
	TeamName              string     `json:"team_name"`
	Score                 int        `json:"score"`
	lastCorrectAnswerTime *time.Time
}

type LeaderboardResponse struct {
	Rows []LeaderboardRow `json:"rows"`
}

func (s *Service) GetLeaderboard(ctx context.Context, questID storage.ID) (*LeaderboardResponse, error) {
	teams, err := s.tms.GetTeams(ctx, &storage.GetTeamsRequest{QuestIDs: []storage.ID{questID}, AcceptedOnly: true})
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, httperrors.Errorf(http.StatusNotFound, "quest %q not found", questID)
		}
		return nil, xerrors.Errorf("get teams: %w", err)
	}
	results, err := s.ah.GetScoreResults(ctx, &storage.GetResultsRequest{QuestID: questID})
	if err != nil {
		return nil, xerrors.Errorf("get results: %w", err)
	}
	penalties, err := s.ah.GetPenalties(ctx, &storage.GetPenaltiesRequest{QuestID: questID})
	if err != nil {
		return nil, xerrors.Errorf("get penalties: %w", err)
	}
	var res LeaderboardResponse
	for _, team := range teams {
		teamScore := results[team.ID]
		teamPenalties := penalties[team.ID]
		teamRes := LeaderboardRow{
			TeamID:   team.ID,
			TeamName: team.Name,
		}
		for _, taskRes := range teamScore {
			teamRes.Score += taskRes.Score
			if teamRes.lastCorrectAnswerTime == nil {
				teamRes.lastCorrectAnswerTime = taskRes.ScoreTime
			} else if teamRes.lastCorrectAnswerTime.Before(*taskRes.ScoreTime) {
				teamRes.lastCorrectAnswerTime = taskRes.ScoreTime
			}
		}
		for _, p := range teamPenalties {
			teamRes.Score -= p.Value
		}
		res.Rows = append(res.Rows, teamRes)
	}
	sort.Slice(res.Rows, func(i, j int) bool {
		if res.Rows[i].Score != res.Rows[j].Score {
			return res.Rows[i].Score >= res.Rows[j].Score
		}
		if res.Rows[i].lastCorrectAnswerTime != nil && res.Rows[j].lastCorrectAnswerTime != nil {
			return res.Rows[i].lastCorrectAnswerTime.Before(*res.Rows[j].lastCorrectAnswerTime)
		}
		return res.Rows[i].TeamName >= res.Rows[j].TeamName
	})
	return &res, nil
}

type TakeHintRequest struct {
	QuestID storage.ID `json:"-"`
	TaskID  storage.ID `json:"task_id"`
	Index   int        `json:"index"`
}

func (s *Service) TakeHint(ctx context.Context, user *storage.User, req *TakeHintRequest) (*storage.Hint, error) {
	team, err := s.tms.GetTeam(ctx, &storage.GetTeamRequest{UserRegistration: &storage.UserRegistration{UserID: user.ID, QuestID: req.QuestID}})
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, httperrors.Errorf(http.StatusNotFound, "team for user %q not found", user.ID)
		}
		return nil, xerrors.Errorf("get team: %w", err)
	}
	if team.RegistrationStatus != storage.RegistrationStatusAccepted {
		return nil, httperrors.New(http.StatusForbidden, "only accepted teams can take hints")
	}
	task, err := s.ts.GetTask(ctx, &storage.GetTaskRequest{
		ID: req.TaskID,
	})
	if err != nil {
		return nil, xerrors.Errorf("get task: %w", err)
	}
	taskGroup, err := s.tgs.GetTaskGroup(ctx, &storage.GetTaskGroupRequest{
		ID:       task.Group.ID,
		TeamData: &storage.TeamData{TeamID: &team.ID},
	})
	if err != nil {
		return nil, xerrors.Errorf("get task group: %w", err)
	}
	now := qtime.Now()
	if team.Quest.QuestType == storage.TypeLinear {
		if taskGroup.TeamInfo == nil {
			return nil, httperrors.Errorf(http.StatusNotAcceptable, "task %q cannot be accessed because task group is closed", req.TaskID)
		}
		if taskGroup.TeamInfo.ClosingTime != nil {
			return nil, httperrors.Errorf(http.StatusNotAcceptable, "task %q is already closed", req.TaskID)
		}
		if taskGroup.HasTimeLimit && taskGroup.TimeLimit != nil {
			deadline := taskGroup.TeamInfo.OpeningTime.Add(time.Duration(*taskGroup.TimeLimit))
			if deadline.Before(now) {
				if _, err = s.tgs.UpsertTeamInfo(ctx, &storage.UpsertTeamInfoRequest{
					TeamID:      team.ID,
					TaskGroupID: taskGroup.ID,
					OpeningTime: taskGroup.TeamInfo.OpeningTime,
					ClosingTime: &deadline,
				}); err != nil {
					logging.Error(ctx, "could not upsert team info", zap.Error(err))
				}
				return nil, httperrors.Errorf(http.StatusNotAcceptable, "task %q deadline exceeded", req.TaskID)
			}
		}
	}

	accepted, err := s.ah.GetAcceptedTasks(ctx, &storage.GetAcceptedTasksRequest{TeamID: team.ID, QuestID: req.QuestID})
	if err != nil {
		return nil, xerrors.Errorf("get results: %w", err)
	}
	if _, ok := accepted[req.TaskID]; ok {
		return nil, httperrors.Errorf(http.StatusNotAcceptable, "question %q already accepted", req.TaskID)
	}
	answerData, err := s.ts.GetAnswerData(ctx, &storage.GetTaskRequest{ID: req.TaskID})
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, httperrors.Errorf(http.StatusNotFound, "task %q not found", req.TaskID)
		}
		return nil, xerrors.Errorf("get answer data: %w", err)
	}
	if len(answerData.FullHints) <= req.Index {
		return nil, httperrors.Errorf(http.StatusBadRequest, "index %d out of hints range", req.Index)
	}
	hint, err := s.ah.TakeHint(ctx, &storage.TakeHintRequest{TeamID: team.ID, TaskID: req.TaskID, Index: req.Index})
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, httperrors.Errorf(http.StatusNotFound, "task %q not found", req.TaskID)
		}
		return nil, xerrors.Errorf("get hint: %w", err)
	}
	return hint, nil
}

type TryAnswerRequest struct {
	QuestID storage.ID `json:"-"`
	TaskID  storage.ID `json:"task_id"`
	Text    string     `json:"text"`
}

type TryAnswerResponse struct {
	Accepted   bool              `json:"accepted"`
	Score      int               `json:"score"`
	Text       string            `json:"text"`
	TaskGroups []AnswerTaskGroup `json:"task_groups,omitempty"`
}

func (s *Service) TryAnswer(ctx context.Context, user *storage.User, req *TryAnswerRequest) (resp *TryAnswerResponse, err error) {
	team, err := s.tms.GetTeam(ctx, &storage.GetTeamRequest{UserRegistration: &storage.UserRegistration{UserID: user.ID, QuestID: req.QuestID}})
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, httperrors.Errorf(http.StatusNotFound, "team for user %q not found", user.ID)
		}
		return nil, xerrors.Errorf("get team: %w", err)
	}
	if team.RegistrationStatus != storage.RegistrationStatusAccepted {
		return nil, httperrors.New(http.StatusForbidden, "only accepted teams can answer tasks")
	}
	acceptedTasks, err := s.ah.GetAcceptedTasks(ctx, &storage.GetAcceptedTasksRequest{TeamID: team.ID, QuestID: req.QuestID})
	if err != nil {
		return nil, xerrors.Errorf("get results: %w", err)
	}
	if acceptedTask, ok := acceptedTasks[req.TaskID]; ok {
		return &TryAnswerResponse{Accepted: true, Text: acceptedTask.Text, Score: acceptedTask.Score}, nil
	}

	answerData, err := s.ts.GetAnswerData(ctx, &storage.GetTaskRequest{ID: req.TaskID})
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, httperrors.Errorf(http.StatusNotFound, "task %q not found", req.TaskID)
		}
		return nil, xerrors.Errorf("get answer data: %w", err)
	}
	taskGroup, err := s.tgs.GetTaskGroup(ctx, &storage.GetTaskGroupRequest{
		ID:           answerData.Group.ID,
		IncludeTasks: true,
		TeamData:     &storage.TeamData{TeamID: &team.ID},
	})
	if err != nil {
		return nil, xerrors.Errorf("get task group: %w", err)
	}
	now := qtime.Now()
	if team.Quest.QuestType == storage.TypeLinear && !taskGroup.Sticky {
		if taskGroup.TeamInfo == nil {
			return nil, httperrors.Errorf(http.StatusNotAcceptable, "task %q cannot be accessed because task group is closed", req.TaskID)
		}
		if taskGroup.TeamInfo.ClosingTime != nil {
			return nil, httperrors.Errorf(http.StatusNotAcceptable, "task %q is already closed", req.TaskID)
		}
		if taskGroup.HasTimeLimit && taskGroup.TimeLimit != nil {
			deadline := taskGroup.TeamInfo.OpeningTime.Add(time.Duration(*taskGroup.TimeLimit))
			if deadline.Before(now) {
				if _, err = s.tgs.UpsertTeamInfo(ctx, &storage.UpsertTeamInfoRequest{
					TeamID:      team.ID,
					TaskGroupID: taskGroup.ID,
					OpeningTime: taskGroup.TeamInfo.OpeningTime,
					ClosingTime: &deadline,
				}); err != nil {
					logging.Error(ctx, "could not upsert team info", zap.Error(err))
				}
				return nil, httperrors.Errorf(http.StatusNotAcceptable, "task %q deadline exceeded", req.TaskID)
			}
		}
	}

	accepted := false
	for _, correctAnswer := range answerData.CorrectAnswers {
		trimmedCorrect := strings.TrimSpace(correctAnswer)
		trimmedAnswer := strings.TrimSpace(req.Text)
		if strings.EqualFold(trimmedCorrect, trimmedAnswer) {
			accepted = true
			break
		}
	}
	tryReq := storage.CreateAnswerTryRequest{
		TaskID: req.TaskID,
		TeamID: team.ID,
		UserID: user.ID,
		Text:   req.Text,
	}

	if !accepted || answerData.Verification == storage.VerificationManual {
		logging.Info(ctx, "answer try",
			zap.Stringer("team_id", team.ID),
			zap.String("team_name", team.Name),
			zap.Stringer("task_id", req.TaskID),
			zap.String("text", req.Text),
		)
		if err = s.ah.CreateAnswerTry(ctx, &tryReq); err != nil {
			return nil, xerrors.Errorf("create answer try: %w", err)
		}
		return &TryAnswerResponse{Accepted: false, Text: req.Text}, nil
	}

	takenHints, err := s.ah.GetHintTakes(ctx, &storage.GetHintTakesRequest{TeamID: team.ID, TaskID: req.TaskID, QuestID: req.QuestID})
	if err != nil {
		return nil, xerrors.Errorf("get hints: %w", err)
	}
	taskHints := takenHints[req.TaskID]
	penalty := 0
	for _, h := range taskHints {
		penalty += h.Hint.Penalty.GetPenaltyPoints(answerData.Reward)
	}
	score := answerData.Reward - penalty
	tryReq.Accepted = true
	tryReq.Score = score
	logging.Info(ctx, "answer try",
		zap.Stringer("team_id", team.ID),
		zap.String("team_name", team.Name),
		zap.Stringer("task_id", req.TaskID),
		zap.String("text", req.Text),
		zap.Int("reward", score),
		zap.Any("taken_hints", taskHints),
	)

	if err = s.ah.CreateAnswerTry(ctx, &tryReq); err != nil {
		return nil, xerrors.Errorf("create answer try: %w", err)
	}
	acceptedTasks[req.TaskID] = storage.AcceptedTask{
		Score: score,
		Text:  req.Text,
	}

	if allSolved(acceptedTasks, taskGroup.Tasks) {
		if _, err = s.tgs.UpsertTeamInfo(ctx, &storage.UpsertTeamInfoRequest{
			TeamID:      team.ID,
			TaskGroupID: taskGroup.ID,
			OpeningTime: taskGroup.TeamInfo.OpeningTime,
			ClosingTime: &now,
		}); err != nil {
			return nil, xerrors.Errorf("upsert team info: %w", err)
		}
	}

	return &TryAnswerResponse{Accepted: true, Text: req.Text, Score: score}, nil
}

func allSolved(accepted storage.AcceptedTasks, tasks []storage.Task) bool {
	for _, task := range tasks {
		if _, ok := accepted[task.ID]; !ok {
			return false
		}
	}
	return true
}

type AddPenaltyRequest struct {
	QuestID storage.ID `json:"-"`
	TeamID  storage.ID `json:"team_id"`
	Penalty int        `json:"penalty"`
}

func (s *Service) AddPenalty(ctx context.Context, req *AddPenaltyRequest) error {
	team, err := s.tms.GetTeam(ctx, &storage.GetTeamRequest{ID: req.TeamID})
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return httperrors.Errorf(http.StatusNotFound, "team %q not found", req.TeamID)
		}
		return xerrors.Errorf("get team: %w", err)
	}
	if team.Quest.ID != req.QuestID {
		return httperrors.Errorf(http.StatusForbidden, "team %q belongs to quest %q", req.TeamID, req.QuestID)
	}
	if err := s.ah.CreatePenalty(ctx, &storage.CreatePenaltyRequest{TeamID: req.TeamID, Penalty: req.Penalty}); err != nil {
		return xerrors.Errorf("create penalty: %w", err)
	}
	return nil
}

type AnswerLog struct {
	TeamID      storage.ID `json:"team_id"`
	Team        string     `json:"team"`
	UserID      storage.ID `json:"user_id,omitempty"`
	User        string     `json:"user,omitempty"`
	TaskGroupID storage.ID `json:"task_group_id"`
	TaskGroup   string     `json:"task_group"`
	TaskID      storage.ID `json:"task_id"`
	Task        string     `json:"task"`
	Accepted    bool       `json:"accepted"`
	Answer      string     `json:"answer"`
	AnswerTime  time.Time  `json:"answer_time"`
}

type AnswerLogResponse struct {
	AnswerLogs    []AnswerLog `json:"answer_logs"`
	TotalPages    int         `json:"total_pages"`
	NextPageToken int64       `json:"next_page_token,omitempty"`
}

func (s *Service) GetAnswerLogs(ctx context.Context, user *storage.User, questID storage.ID, opts ...storage.FilteringOption) (AnswerLogResponse, error) {
	logResp, err := s.ah.GetAnswerTries(ctx, &storage.GetAnswerTriesRequest{QuestID: questID}, opts...)
	if err != nil {
		return AnswerLogResponse{}, xerrors.Errorf("get answer tries: %w", err)
	}
	logs := make([]AnswerLog, 0, len(logResp.AnswerLogs))
	for _, log := range logResp.AnswerLogs {
		al := AnswerLog{
			TeamID:      log.Team.ID,
			Team:        log.Team.Name,
			TaskGroupID: log.TaskGroup.ID,
			TaskGroup:   log.TaskGroup.Name,
			TaskID:      log.Task.ID,
			Task:        log.Task.Name,
			Accepted:    log.Accepted,
			Answer:      log.Answer,
			AnswerTime:  log.AnswerTime,
		}
		if log.User != nil {
			al.UserID = log.User.ID
			al.User = log.User.Username
		}
		logs = append(logs, al)
	}
	resp := AnswerLogResponse{
		AnswerLogs:    logs,
		TotalPages:    logResp.TotalPages,
		NextPageToken: logResp.NextToken,
	}
	return resp, nil
}
