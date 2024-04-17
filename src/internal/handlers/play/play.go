package play

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/xerrors"

	pgdb "questspace/internal/pgdb/client"
	"questspace/internal/questspace/quests"
	"questspace/pkg/application/httperrors"
	"questspace/pkg/auth/jwt"
	"questspace/pkg/dbnode"
	"questspace/pkg/storage"
)

type Handler struct {
	clientFactory pgdb.QuestspaceClientFactory
}

func NewHandler(clientFactory pgdb.QuestspaceClientFactory) *Handler {
	return &Handler{
		clientFactory: clientFactory,
	}
}

type GetResponse struct {
	Quest      *storage.Quest      `json:"quest"`
	TaskGroups []storage.TaskGroup `json:"task_groups"`
	Team       *storage.Team       `json:"team"`
}

// HandleGet handles GET quest/:id/play request
//
// @Summary	Get task groups with tasks for play-mode
// @Tags	PlayMode
// @Param	quest_id	path		string		true	"Quest ID"
// @Success	200			{object}	play.GetResponse
// @Failure	400
// @Failure	401
// @Failure 404
// @Failure 406
// @Router	/quest/{id}/play [get]
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
	quests.SetStatus(quest)
	if quest.Status != storage.StatusRunning {
		return httperrors.New(http.StatusNotAcceptable, "cannot get tasks before quest start")
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
