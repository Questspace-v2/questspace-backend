package play

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/xerrors"

	"questspace/internal/handlers/transport"
	pgdb "questspace/internal/pgdb/client"
	"questspace/internal/questspace/game"
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
// @Summary		Get task groups with tasks for play-mode
// @Tags		PlayMode
// @Param		quest_id	path		string		true	"Quest ID"
// @Success		200			{object}	game.AnswerDataResponse
// @Failure		400
// @Failure		401
// @Failure 	404
// @Failure 	406
// @Router		/quest/{id}/play [get]
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

	service := game.NewService(s, s, s, s)
	req := game.AnswerDataRequest{Quest: quest, Team: userTeam, TaskGroups: taskGroups}
	resp, err := service.FillAnswerData(c, &req)
	if err != nil {
		return xerrors.Errorf("fill answer data: %w", err)
	}

	c.JSON(http.StatusOK, resp)
	return nil
}

type TakeHintRequest struct {
	TaskID string `json:"task_id"`
	Index  int    `json:"index"`
}

// HandleTakeHint handles POST quest/:id/hint request
//
// @Summary		Take hint for task in play-mode
// @Tags		PlayMode
// @Param		quest_id	path		string					true	"Quest ID"
// @Param		request		body		play.TakeHintRequest	true	"Take hint request"
// @Success		200			{object}	storage.Hint
// @Failure		400
// @Failure		401
// @Failure 	404
// @Failure 	406
// @Router		/quest/{id}/hint [post]
// @Security 	ApiKeyAuth
func (h *Handler) HandleTakeHint(c *gin.Context) error {
	questID := c.Param("id")
	req, err := transport.UnmarshalRequestData[TakeHintRequest](c.Request)
	if err != nil {
		return xerrors.Errorf("%w", err)
	}
	uauth, err := jwt.GetUserFromContext(c)
	if err != nil {
		return xerrors.Errorf("%w", err)
	}

	s, tx, err := h.clientFactory.NewStorageTx(c, nil)
	if err != nil {
		return xerrors.Errorf("start tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	quest, err := s.GetQuest(c, &storage.GetQuestRequest{ID: questID})
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return httperrors.Errorf(http.StatusNotFound, "quest %q not found", questID)
		}
		return xerrors.Errorf("get quest: %w", err)
	}
	quests.SetStatus(quest)
	if quest.Status != storage.StatusRunning {
		return httperrors.New(http.StatusNotAcceptable, "cannot take hints before quest start")
	}

	srv := game.NewService(s, s, s, s)
	srvReq := game.TakeHintRequest{QuestID: questID, TaskID: req.TaskID, Index: req.Index}
	hint, err := srv.TakeHint(c, uauth, &srvReq)
	if err != nil {
		return xerrors.Errorf("hint error: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return xerrors.Errorf("commit tx: %w", err)
	}

	c.JSON(http.StatusOK, hint)
	return nil
}

type TryAnswerRequest struct {
	TaskID string
	Text   string
}

// HandleTryAnswer handles POST quest/:id/answer request
//
// @Summary		Answer task in play-mode
// @Tags		PlayMode
// @Param		quest_id	path		string					true	"Quest ID"
// @Param		request		body		play.TryAnswerRequest	true	"Answer data"
// @Success		200			{object}	game.TryAnswerResponse
// @Failure		400
// @Failure		401
// @Failure 	404
// @Failure 	406
// @Router		/quest/{id}/answer [post]
// @Security 	ApiKeyAuth
func (h *Handler) HandleTryAnswer(c *gin.Context) error {
	questID := c.Param("id")
	req, err := transport.UnmarshalRequestData[TryAnswerRequest](c.Request)
	if err != nil {
		return xerrors.Errorf("%w", err)
	}
	uauth, err := jwt.GetUserFromContext(c)
	if err != nil {
		return xerrors.Errorf("%w", err)
	}

	s, tx, err := h.clientFactory.NewStorageTx(c, nil)
	if err != nil {
		return xerrors.Errorf("start tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	quest, err := s.GetQuest(c, &storage.GetQuestRequest{ID: questID})
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return httperrors.Errorf(http.StatusNotFound, "quest %q not found", questID)
		}
		return xerrors.Errorf("get quest: %w", err)
	}
	quests.SetStatus(quest)
	if quest.Status != storage.StatusRunning {
		return httperrors.New(http.StatusNotAcceptable, "cannot take hints before quest start")
	}

	srv := game.NewService(s, s, s, s)
	srvReq := game.TryAnswerRequest{TaskID: req.TaskID, Text: req.Text, QuestID: questID}
	try, err := srv.TryAnswer(c, uauth, &srvReq)
	if err != nil {
		return xerrors.Errorf("try answer: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return xerrors.Errorf("commit tx: %w", err)
	}

	c.JSON(http.StatusOK, try)
	return nil
}

// HandleGetTableResults handles GET quest/:id/table request
//
// @Summary		Get leaderboard table during quest
// @Tags		PlayMode
// @Param		quest_id	path		string		true	"Quest ID"
// @Success		200			{object}	game.TeamResults
// @Failure		400
// @Failure		401
// @Failure 	403
// @Failure 	404
// @Router		/quest/{id}/table [get]
// @Security 	ApiKeyAuth
func (h *Handler) HandleGetTableResults(c *gin.Context) error {
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
			return httperrors.Errorf(http.StatusNotFound, "quest %q not found", questID)
		}
		return xerrors.Errorf("get quest: %w", err)
	}
	if quest.Creator.ID != uauth.ID {
		return httperrors.New(http.StatusForbidden, "only creator can view leaderboard during quest time")
	}

	srv := game.NewService(s, s, s, s)
	leaderBoard, err := srv.GetResults(c, questID)
	if err != nil {
		return xerrors.Errorf("get results: %w", err)
	}

	c.JSON(http.StatusOK, leaderBoard)
	return nil
}
