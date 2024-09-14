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

	"go.uber.org/zap"
	"golang.org/x/xerrors"

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
	Taken bool   `json:"taken"`
	Text  string `json:"text,omitempty"`
}

type AnswerTask struct {
	ID           string                   `json:"id"`
	OrderIdx     int                      `json:"order_idx"`
	Name         string                   `json:"name"`
	Question     string                   `json:"question"`
	Reward       int                      `json:"reward"`
	Verification storage.VerificationType `json:"verification_type" enums:"auto,manual"`
	Hints        []AnswerTaskHint         `json:"hints"`
	Accepted     bool                     `json:"accepted"`
	Answer       string                   `json:"answer,omitempty"`
	PubTime      *time.Time               `json:"pub_time,omitempty"`
	MediaLink    string                   `json:"media_link"`
}

type AnswerTaskGroup struct {
	ID       string       `json:"id"`
	OrderIdx int          `json:"order_idx"`
	Name     string       `json:"name"`
	PubTime  *time.Time   `json:"pub_time,omitempty"`
	Tasks    []AnswerTask `json:"tasks"`
}

type AnswerDataResponse struct {
	Quest      *storage.Quest    `json:"quest"`
	Team       *storage.Team     `json:"team"`
	TaskGroups []AnswerTaskGroup `json:"task_groups"`
}

func (s *Service) FillAnswerData(ctx context.Context, req *AnswerDataRequest) (*AnswerDataResponse, error) {
	tookHints, err := s.ah.GetHintTakes(ctx, &storage.GetHintTakesRequest{TeamID: req.Team.ID, QuestID: req.Quest.ID})
	if err != nil {
		return nil, xerrors.Errorf("get hints: %w", err)
	}
	acceptedTasks, err := s.ah.GetAcceptedTasks(ctx, &storage.GetAcceptedTasksRequest{TeamID: req.Team.ID, QuestID: req.Quest.ID})
	if err != nil {
		return nil, xerrors.Errorf("get accepted tasks: %w", err)
	}

	taskGroups := make([]AnswerTaskGroup, 0, len(req.TaskGroups))
	for _, tg := range req.TaskGroups {
		newTg := AnswerTaskGroup{
			ID:       tg.ID,
			OrderIdx: tg.OrderIdx,
			Name:     tg.Name,
			PubTime:  tg.PubTime,
			Tasks:    make([]AnswerTask, 0, len(tg.Tasks)),
		}

		for _, t := range tg.Tasks {
			newT := AnswerTask{
				ID:           t.ID,
				OrderIdx:     t.OrderIdx,
				Name:         t.Name,
				Question:     t.Question,
				Reward:       t.Reward,
				Verification: t.Verification,
				Hints:        make([]AnswerTaskHint, len(t.Hints)),
				PubTime:      t.PubTime,
				MediaLink:    t.MediaLink,
			}
			if ans, ok := acceptedTasks[t.ID]; ok {
				newT.Accepted = true
				newT.Answer = ans
			}
			for _, h := range tookHints[newT.ID] {
				newT.Hints[h.Hint.Index].Taken = true
				newT.Hints[h.Hint.Index].Text = h.Hint.Text
			}
			newTg.Tasks = append(newTg.Tasks, newT)
		}
		taskGroups = append(taskGroups, newTg)
	}

	resp := &AnswerDataResponse{
		Quest:      req.Quest,
		Team:       req.Team,
		TaskGroups: taskGroups,
	}
	return resp, nil
}

type TaskResult struct {
	Score      int
	groupIndex int
	taskIndex  int
}

type TeamResult struct {
	TeamID                string
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

func (s *Service) GetResults(ctx context.Context, questID string) (*TeamResults, error) {
	teams, err := s.tms.GetTeams(ctx, &storage.GetTeamsRequest{QuestIDs: []string{questID}})
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
	TeamID                string `json:"team_id"`
	TeamName              string `json:"team_name"`
	Score                 int    `json:"score"`
	lastCorrectAnswerTime *time.Time
}

type LeaderboardResponse struct {
	Rows []LeaderboardRow `json:"rows"`
}

func (s *Service) GetLeaderboard(ctx context.Context, questID string) (*LeaderboardResponse, error) {
	teams, err := s.tms.GetTeams(ctx, &storage.GetTeamsRequest{QuestIDs: []string{questID}})
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
	QuestID string `json:"-"`
	TaskID  string `json:"task_id"`
	Index   int    `json:"index"`
}

func (s *Service) TakeHint(ctx context.Context, user *storage.User, req *TakeHintRequest) (*storage.Hint, error) {
	team, err := s.tms.GetTeam(ctx, &storage.GetTeamRequest{UserRegistration: &storage.UserRegistration{UserID: user.ID, QuestID: req.QuestID}})
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, httperrors.Errorf(http.StatusNotFound, "team for user %q not found", user.ID)
		}
		return nil, xerrors.Errorf("get team: %w", err)
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
	if len(answerData.Hints) <= req.Index {
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
	QuestID string `json:"-"`
	TaskID  string `json:"task_id"`
	Text    string `json:"text"`
}

type TryAnswerResponse struct {
	Accepted bool   `json:"accepted"`
	Score    int    `json:"score,omitempty"`
	Text     string `json:"text"`
}

func (s *Service) TryAnswer(ctx context.Context, user *storage.User, req *TryAnswerRequest) (*TryAnswerResponse, error) {
	team, err := s.tms.GetTeam(ctx, &storage.GetTeamRequest{UserRegistration: &storage.UserRegistration{UserID: user.ID, QuestID: req.QuestID}})
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, httperrors.Errorf(http.StatusNotFound, "team for user %q not found", user.ID)
		}
		return nil, xerrors.Errorf("get team: %w", err)
	}
	acceptedTasks, err := s.ah.GetAcceptedTasks(ctx, &storage.GetAcceptedTasksRequest{TeamID: team.ID, QuestID: req.QuestID})
	if err != nil {
		return nil, xerrors.Errorf("get results: %w", err)
	}
	if acceptedText, ok := acceptedTasks[req.TaskID]; ok {
		return &TryAnswerResponse{Accepted: true, Text: acceptedText}, nil
	}

	answerData, err := s.ts.GetAnswerData(ctx, &storage.GetTaskRequest{ID: req.TaskID})
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, httperrors.Errorf(http.StatusNotFound, "task %q not found", req.TaskID)
		}
		return nil, xerrors.Errorf("get answer data: %w", err)
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
		Text:   req.Text,
	}

	logging.Info(ctx, "answer try",
		zap.String("team_id", team.ID),
		zap.String("team_name", team.Name),
		zap.String("task_id", req.TaskID),
		zap.String("text", req.Text),
	)

	if !accepted || answerData.Verification == storage.VerificationManual {
		if err = s.ah.CreateAnswerTry(ctx, &tryReq); err != nil {
			return nil, xerrors.Errorf("create answer try: %w", err)
		}
		return &TryAnswerResponse{Accepted: false, Text: req.Text}, nil
	}

	tookHints, err := s.ah.GetHintTakes(ctx, &storage.GetHintTakesRequest{TeamID: team.ID, TaskID: req.TaskID, QuestID: req.QuestID})
	if err != nil {
		return nil, xerrors.Errorf("get hints: %w", err)
	}
	taskHints := tookHints[req.TaskID]
	score := answerData.Reward * (5 - len(taskHints)) / 5
	tryReq.Accepted = true
	tryReq.Score = score
	if err = s.ah.CreateAnswerTry(ctx, &tryReq); err != nil {
		return nil, xerrors.Errorf("create answer try: %w", err)
	}

	return &TryAnswerResponse{Accepted: true, Text: req.Text, Score: score}, nil
}

type AddPenaltyRequest struct {
	QuestID string `json:"-"`
	TeamID  string `json:"team_id"`
	Penalty int    `json:"penalty"`
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
