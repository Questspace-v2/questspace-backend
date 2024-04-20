package game

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"
	"golang.org/x/xerrors"

	"questspace/pkg/application/httperrors"
	"questspace/pkg/application/logging"
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
	TaskID   string `json:"id"`
	TaskName string `json:"name"`
	Score    int    `json:"score"`
}

type TaskGroupResult struct {
	GroupID   string       `json:"id"`
	GroupName string       `json:"name"`
	Tasks     []TaskResult `json:"tasks"`
}

type TeamResult struct {
	TeamID                string            `json:"id"`
	TeamName              string            `json:"name"`
	TotalScore            int               `json:"total_score"`
	TaskGroups            []TaskGroupResult `json:"task_groups"`
	lastCorrectAnswerTime *time.Time
}

type TeamResults struct {
	Results []TeamResult `json:"results"`
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

	var res TeamResults
	for _, team := range teams {
		teamScore := results[team.ID]
		teamRes := TeamResult{
			TeamID:   team.ID,
			TeamName: team.Name,
		}
		for _, tg := range taskGroups {
			tgRes := TaskGroupResult{
				GroupID:   tg.ID,
				GroupName: tg.Name,
			}
			for _, task := range tg.Tasks {
				taskRes := TaskResult{
					TaskID:   task.ID,
					TaskName: task.Name,
				}
				if scoreRes, ok := teamScore[task.ID]; ok {
					taskRes.Score = scoreRes.Score
					teamRes.TotalScore += scoreRes.Score
					if teamRes.lastCorrectAnswerTime == nil {
						teamRes.lastCorrectAnswerTime = scoreRes.ScoreTime
					} else if teamRes.lastCorrectAnswerTime.Before(*scoreRes.ScoreTime) {
						teamRes.lastCorrectAnswerTime = scoreRes.ScoreTime
					}
				}
				tgRes.Tasks = append(tgRes.Tasks, taskRes)
			}
			teamRes.TaskGroups = append(teamRes.TaskGroups, tgRes)
		}
		res.Results = append(res.Results, teamRes)
	}

	sort.Slice(res.Results, func(i, j int) bool {
		if res.Results[i].TotalScore != res.Results[j].TotalScore {
			return res.Results[i].TotalScore < res.Results[j].TotalScore
		}
		if res.Results[i].lastCorrectAnswerTime != nil && res.Results[j].lastCorrectAnswerTime != nil {
			return res.Results[i].lastCorrectAnswerTime.After(*res.Results[j].lastCorrectAnswerTime)
		}
		return res.Results[i].TeamName < res.Results[j].TeamName
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
		if err := s.ah.CreateAnswerTry(ctx, &tryReq); err != nil {
			return nil, xerrors.Errorf("create answer try: %w", err)
		}
		return &TryAnswerResponse{Accepted: false, Text: req.Text}, nil
	}

	tookHints, err := s.ah.GetHintTakes(ctx, &storage.GetHintTakesRequest{TeamID: team.ID, TaskID: req.TaskID, QuestID: req.QuestID})
	if err != nil {
		return nil, xerrors.Errorf("get hints: %w", err)
	}
	score := answerData.Reward * (5 - len(tookHints)) / 5
	tryReq.Accepted = true
	tryReq.Score = score
	if err := s.ah.CreateAnswerTry(ctx, &tryReq); err != nil {
		return nil, xerrors.Errorf("create answer try: %w", err)
	}

	return &TryAnswerResponse{Accepted: true, Text: req.Text, Score: score}, nil
}
