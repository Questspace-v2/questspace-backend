package taskgroups

import (
	"context"
	"errors"
	"net/http"

	"golang.org/x/xerrors"

	"questspace/internal/pgdb"
	"questspace/internal/questspace/quests"
	"questspace/internal/questspace/taskgroups"
	"questspace/internal/questspace/taskgroups/requests"
	"questspace/internal/questspace/tasks"
	"questspace/pkg/auth/jwt"
	"questspace/pkg/dbnode"
	"questspace/pkg/httperrors"
	"questspace/pkg/storage"
	"questspace/pkg/transport"
)

type Handler struct {
	clientFactory pgdb.QuestspaceClientFactory
}

func NewHandler(c pgdb.QuestspaceClientFactory) *Handler {
	return &Handler{clientFactory: c}
}

type TaskGroups []*storage.TaskGroup

// HandleBulkUpdate handles PATCH quest/:id/task-groups/bulk request
//
// @Summary		Patch task groups by creating new ones, delete, update and reorder all ones. Returns all exising task groups.
// @Tags		TaskGroups
// @Param		request	body		storage.TaskGroupsBulkUpdateRequest	true	"Requests to delete/create/update task groups"
// @Success		200		{object}	requests.CreateFullResponse
// @Failure		400
// @Failure		401
// @Failure		403
// @Router		/quest/{id}/task-groups/bulk [patch]
// @Security 	ApiKeyAuth
func (h *Handler) HandleBulkUpdate(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	questID, err := transport.UUIDParam(r, "id")
	if err != nil {
		return xerrors.Errorf("%w", err)
	}
	req, err := transport.UnmarshalRequestData[storage.TaskGroupsBulkUpdateRequest](r)
	if err != nil {
		return xerrors.Errorf("%w", err)
	}
	req.QuestID = questID

	uauth, err := jwt.GetUserFromContext(ctx)
	if err != nil {
		return xerrors.Errorf("%w", err)
	}

	s, tx, err := h.clientFactory.NewStorageTx(ctx, nil)
	if err != nil {
		return xerrors.Errorf("get storage client: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	quest, err := s.GetQuest(ctx, &storage.GetQuestRequest{ID: questID})
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return httperrors.Errorf(http.StatusNotFound, "not found quest %q", questID)
		}
		return xerrors.Errorf("get quest: %w", err)
	}

	if quest.Creator.ID != uauth.ID {
		return httperrors.Errorf(http.StatusForbidden, "cannot change others' quests")
	}

	taskUpdater := tasks.NewUpdater(s)
	updater := taskgroups.NewUpdater(s, taskUpdater)
	tasksGroups, err := updater.BulkUpdateTaskGroups(ctx, req)
	if err != nil {
		return xerrors.Errorf("bulk update: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return xerrors.Errorf("commit tx: %w", err)
	}

	resp := requests.CreateFullResponse{TaskGroups: tasksGroups}
	if err = transport.ServeJSONResponse(w, http.StatusOK, resp); err != nil {
		return err
	}
	return nil
}

// HandleCreate handles POST quest/:id/task-groups request
//
// @Summary		[Deprecated] Create task groups and tasks. All previously created task groups and tasks would be deleted and overridden.
// @Tags		TaskGroups
// @Param		quest_id	path		string						true	"Quest ID"
// @Param		request		body		requests.CreateFullRequest	true	"All task groups with inner tasks to create"
// @Success		200			{object}	requests.CreateFullResponse
// @Failure		400
// @Failure		401
// @Failure		403
// @Failure 	404
// @Router		/quest/{id}/task-groups [post]
// @Security 	ApiKeyAuth
func (h *Handler) HandleCreate(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	questID, err := transport.UUIDParam(r, "id")
	if err != nil {
		return xerrors.Errorf("%w", err)
	}
	req, err := transport.UnmarshalRequestData[requests.CreateFullRequest](r)
	if err != nil {
		return xerrors.Errorf("%w", err)
	}
	req.QuestID = questID

	uauth, err := jwt.GetUserFromContext(ctx)
	if err != nil {
		return xerrors.Errorf("%w", err)
	}

	s, tx, err := h.clientFactory.NewStorageTx(ctx, nil)
	if err != nil {
		return xerrors.Errorf("start tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	q, err := s.GetQuest(ctx, &storage.GetQuestRequest{ID: questID})
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return httperrors.Errorf(http.StatusNotFound, "not found quest %q", questID)
		}
		return xerrors.Errorf("get quest: %w", err)
	}
	if q.Creator.ID != uauth.ID {
		return httperrors.Errorf(http.StatusForbidden, "cannot change others' quests")
	}
	quests.SetStatus(q)
	if q.Status == storage.StatusRunning || q.Status == storage.StatusWaitResults || q.Status == storage.StatusFinished {
		return httperrors.Errorf(http.StatusForbidden, "do not use create method when quest is already running")
	}
	serv := taskgroups.NewService(s, s)
	resp, err := serv.Create(ctx, req)
	if err != nil {
		return xerrors.Errorf("create taskgroups: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return xerrors.Errorf("commit tx: %w", err)
	}
	if err = transport.ServeJSONResponse(w, http.StatusOK, resp); err != nil {
		return err
	}
	return nil
}

type GetResponse struct {
	Quest      *storage.Quest      `json:"quest"`
	TaskGroups []storage.TaskGroup `json:"task_groups"`
}

// HandleGet handles GET quest/:id/task-groups request
//
// @Summary		Get task groups with tasks for quest creator
// @Tags		TaskGroups
// @Param		quest_id	path		string		true	"Quest ID"
// @Success		200			{object}	taskgroups.GetResponse
// @Failure		400
// @Failure		401
// @Failure		403
// @Failure 	404
// @Failure 	406
// @Router		/quest/{id}/task-groups [get]
// @Security 	ApiKeyAuth
func (h *Handler) HandleGet(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	questID, err := transport.UUIDParam(r, "id")
	if err != nil {
		return xerrors.Errorf("%w", err)
	}
	uauth, err := jwt.GetUserFromContext(ctx)
	if err != nil {
		return xerrors.Errorf("%w", err)
	}

	s, err := h.clientFactory.NewStorage(ctx, dbnode.Alive)
	if err != nil {
		return xerrors.Errorf("get storage: %w", err)
	}

	quest, err := s.GetQuest(ctx, &storage.GetQuestRequest{ID: questID})
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return httperrors.Errorf(http.StatusNotFound, "not found quest %q", questID)
		}
		return xerrors.Errorf("get quest: %w", err)
	}
	if quest.Creator.ID != uauth.ID {
		return httperrors.Errorf(http.StatusForbidden, "only creator can get tasks outside of playmode", questID)
	}
	quests.SetStatus(quest)
	taskGroups, err := s.GetTaskGroups(ctx, &storage.GetTaskGroupsRequest{QuestID: questID, IncludeTasks: true})
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return httperrors.Errorf(http.StatusNotFound, "not found quest %q", questID)
		}
		return xerrors.Errorf("get taskgroups: %w", err)
	}
	resp := GetResponse{Quest: quest, TaskGroups: taskGroups}

	if err = transport.ServeJSONResponse(w, http.StatusOK, &resp); err != nil {
		return err
	}
	return nil
}
