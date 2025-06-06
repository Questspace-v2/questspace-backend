package taskgroups

import (
	"context"
	"errors"
	"net/http"

	"github.com/yandex/perforator/library/go/core/xerrors"

	"questspace/internal/questspace/taskgroups/requests"
	"questspace/internal/questspace/tasks"
	"questspace/pkg/httperrors"
	"questspace/pkg/storage"
)

type Service struct {
	tg      storage.TaskGroupStorage
	updater *Updater
}

func NewService(tg storage.TaskGroupStorage, ts storage.TaskStorage, v requests.ImageValidator) *Service {
	upd := NewUpdater(tg, tasks.NewUpdater(ts), v)
	return &Service{
		tg:      tg,
		updater: upd,
	}
}

func (s *Service) Create(ctx context.Context, req *requests.CreateFullRequest) (requests.CreateFullResponse, error) {
	old, err := s.tg.GetTaskGroups(ctx, &storage.GetTaskGroupsRequest{QuestID: req.QuestID})
	if err != nil {
		return requests.CreateFullResponse{}, xerrors.Errorf("get task groups: %w", err)
	}
	if len(old) > 0 {
		var errs []error
		for _, tg := range old {
			if err := s.tg.DeleteTaskGroup(ctx, &storage.DeleteTaskGroupRequest{ID: tg.ID}); err != nil {
				errs = append(errs, xerrors.Errorf("delete task group %q: %w", tg.ID, err))
			}
		}
		if len(errs) > 0 {
			return requests.CreateFullResponse{}, xerrors.Errorf("delete old task groups: %w", errors.Join(errs...))
		}
	}

	bulkRequest := storage.TaskGroupsBulkUpdateRequest{QuestID: req.QuestID}
	for i, tg := range req.TaskGroups {
		createReq := storage.CreateTaskGroupRequest{
			QuestID:      req.QuestID,
			Name:         tg.Name,
			Description:  tg.Description,
			OrderIdx:     i,
			Sticky:       tg.Sticky,
			PubTime:      tg.PubTime,
			HasTimeLimit: tg.HasTimeLimit,
			TimeLimit:    tg.TimeLimit,
		}

		for j, t := range tg.Tasks {
			if string(t.Verification) == "" {
				t.Verification = storage.VerificationAuto
			}
			if len(t.Hints) > 3 {
				return requests.CreateFullResponse{}, httperrors.Errorf(http.StatusBadRequest, "only 3 or less hints allowed")
			}
			createReq.Tasks = append(createReq.Tasks, storage.CreateTaskRequest{
				OrderIdx:       j,
				Name:           t.Name,
				Question:       t.Question,
				MediaLink:      t.MediaLink,
				MediaLinks:     t.MediaLinks,
				Reward:         t.Reward,
				CorrectAnswers: t.CorrectAnswers,
				Hints:          t.Hints,
				FullHints:      t.FullHints,
				Verification:   t.Verification,
			})
		}
		bulkRequest.Create = append(bulkRequest.Create, createReq)
	}

	taskGroups, err := s.updater.BulkUpdateTaskGroups(ctx, &bulkRequest)
	if err != nil {
		return requests.CreateFullResponse{}, xerrors.Errorf("update task groups: %w", err)
	}
	resp := requests.CreateFullResponse{TaskGroups: taskGroups}
	return resp, nil
}
