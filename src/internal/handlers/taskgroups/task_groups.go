package taskgroups

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/xerrors"

	"questspace/internal/handlers/transport"
	pgdb "questspace/internal/pgdb/client"
	"questspace/internal/questspace/taskgroups"
	"questspace/internal/questspace/taskgroups/requests"
	"questspace/pkg/application/httperrors"
	"questspace/pkg/auth/jwt"
	"questspace/pkg/dbnode"
	"questspace/pkg/storage"
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
// @Summary	[WIP] Patch task groups by creating new ones, delete, update and reorder all ones. Returns all exising task groups.
// @Tags	TaskGroups
// @Param	request	body		storage.TaskGroupsBulkUpdateRequest	true	"Requests to delete/create/update task groups"
// @Success	200		{object}	TaskGroups
// @Failure	400
// @Failure	401
// @Failure	403
// @Router	/quest/{id}/task-groups/bulk [patch]
func (h *Handler) HandleBulkUpdate(c *gin.Context) error {
	c.Status(http.StatusNotImplemented)
	return nil

	//nolint:govet
	//TODO(svayp11): add auth
	questID := c.Param("id")
	req, err := transport.UnmarshalRequestData[storage.TaskGroupsBulkUpdateRequest](c.Request)
	if err != nil {
		return xerrors.Errorf("%w", err)
	}
	req.QuestID = questID

	s, tx, err := h.clientFactory.NewStorageTx(c, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return xerrors.Errorf("get storage client: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	updater := taskgroups.NewUpdater(s, nil)
	tasksGroups, err := updater.BulkUpdateTaskGroups(c, req)
	if err != nil {
		return xerrors.Errorf("bulk update: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return xerrors.Errorf("commit tx: %w", err)
	}

	c.JSON(http.StatusOK, tasksGroups)
	return nil
}

// HandleCreate handles POST quest/:id/task-groups request
//
// @Summary	Create task groups and tasks. All previously created task groups and tasks would be deleted and overridden.
// @Tags	TaskGroups
// @Param	quest_id	path		string							true	"Quest ID"
// @Param	request		body		requests.CreateFullRequest	true	"All task groups with inner tasks to create"
// @Success	200			{object}	requests.CreateFullResponse
// @Failure	400
// @Failure	401
// @Failure	403
// @Failure 404
// @Router	/quest/{id}/task-groups [post]
// @Security 	ApiKeyAuth
func (h *Handler) HandleCreate(c *gin.Context) error {
	questID := c.Param("id")
	req, err := transport.UnmarshalRequestData[requests.CreateFullRequest](c.Request)
	if err != nil {
		return xerrors.Errorf("%w", err)
	}
	req.QuestID = questID

	uauth, err := jwt.GetUserFromContext(c)
	if err != nil {
		return xerrors.Errorf("%w", err)
	}

	s, tx, err := h.clientFactory.NewStorageTx(c, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return xerrors.Errorf("start tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	q, err := s.GetQuest(c, &storage.GetQuestRequest{ID: questID})
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return httperrors.Errorf(http.StatusNotFound, "not found quest %q", questID)
		}
		return xerrors.Errorf("get quest: %w", err)
	}
	if q.Creator.ID != uauth.ID {
		return httperrors.Errorf(http.StatusForbidden, "cannot change others' quests")
	}
	serv := taskgroups.NewService(s, s)
	resp, err := serv.Create(c, req)
	if err != nil {
		return xerrors.Errorf("create taskgroups: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return xerrors.Errorf("commit tx: %w", err)
	}
	c.JSON(http.StatusOK, resp)
	return nil
}

type GetResponse struct {
	Quest      *storage.Quest      `json:"quest"`
	TaskGroups []storage.TaskGroup `json:"task_groups"`
	Team       *storage.Team       `json:"team"`
}

// HandleGet handles GET quest/:id/task-groups request
//
// @Summary	Get task groups with tasks
// @Tags	TaskGroups
// @Param	quest_id	path		string		true	"Quest ID"
// @Success	200			{object}	taskgroups.GetResponse
// @Failure	400
// @Failure	401
// @Failure	403
// @Failure 404
// @Failure 406
// @Router	/quest/{id}/task-groups [get]
// @Security 	ApiKeyAuth
func (h *Handler) HandleGet(c *gin.Context) error {
	questID := c.Param("id")
	uauth, err := jwt.GetUserFromContext(c)
	if err != nil {
		return xerrors.Errorf("%w", err)
	}

	s, err := h.clientFactory.NewStorage(c, dbnode.Alive)
	if err != nil {
		return xerrors.Errorf("get storage: %w", err)
	}

	quest, err := s.GetQuest(c, &storage.GetQuestRequest{ID: questID})
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return httperrors.Errorf(http.StatusNotFound, "not found quest %q", questID)
		}
		return xerrors.Errorf("get quest: %w", err)
	}
	taskGroups, err := s.GetTaskGroups(c, &storage.GetTaskGroupsRequest{QuestID: questID, IncludeTasks: true})
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return httperrors.Errorf(http.StatusNotFound, "not found quest %q", questID)
		}
		return xerrors.Errorf("get taskgroups: %w", err)
	}
	userTeam, err := s.GetTeam(c, &storage.GetTeamRequest{UserRegistration: &storage.UserRegistration{UserID: uauth.ID, QuestID: questID}})
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return httperrors.Errorf(http.StatusNotAcceptable, "user %q has no team", uauth.ID)
		}
		return xerrors.Errorf("get team: %w", err)
	}
	resp := GetResponse{Quest: quest, TaskGroups: taskGroups, Team: userTeam}

	c.JSON(http.StatusOK, resp)
	return nil
}
