package game

import (
	"context"
	"errors"
	"net/http"
	"time"

	"golang.org/x/xerrors"

	"questspace/pkg/application/httperrors"
	"questspace/pkg/storage"
)

type AHStorage interface {
	storage.AnswerStorage
	storage.HintStorage
}

type Service struct {
	ts  storage.TaskStorage
	tms storage.TeamStorage
	ah  AHStorage
}

func NewService(ts storage.TaskStorage, tms storage.TeamStorage, ah AHStorage) *Service {
	return &Service{
		ts:  ts,
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
			if _, ok := acceptedTasks[t.ID]; ok {
				newT.Accepted = true
			}
			for _, h := range tookHints[newT.ID] {
				newT.Hints[h.Hint.Index].Taken = true
				newT.Hints[h.Hint.Index].Text = h.Hint.Text
			}
			newTg.Tasks = append(newTg.Tasks, newT)
		}
	}

	resp := &AnswerDataResponse{
		Quest:      req.Quest,
		Team:       req.Team,
		TaskGroups: taskGroups,
	}
	return resp, nil
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
		if req.Text == correctAnswer {
			accepted = true
			break
		}
	}
	tryReq := storage.CreateAnswerTryRequest{
		TaskID: req.TaskID,
		TeamID: team.ID,
		Text:   req.Text,
	}
	if !accepted {
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
