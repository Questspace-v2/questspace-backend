package quest

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/xerrors"

	pgdb "questspace/internal/pgdb/client"
	"questspace/internal/validate"
	aerrors "questspace/pkg/application/errors"
	"questspace/pkg/application/logging"
	"questspace/pkg/auth/jwt"
	"questspace/pkg/dbnode"
	"questspace/pkg/storage"
)

type Handler struct {
	clientFactory pgdb.QuestspaceClientFactory
	fetcher       http.Client
}

func NewHandler(cf pgdb.QuestspaceClientFactory, f http.Client) *Handler {
	return &Handler{
		clientFactory: cf,
		fetcher:       f,
	}
}

// HandleCreate handles POST /quest request
//
//		@Summary	Create new quest
//		@Param		request	body		storage.CreateQuestRequest	true	"Main quest information"
//		@Success	200		{object}	storage.Quest
//		@Failure	400
//	    @Failure    401
//		@Failure	415
//		@Router		/quest [post]
func (h *Handler) HandleCreate(c *gin.Context) error {
	data, err := c.GetRawData()
	if err != nil {
		return xerrors.Errorf("failed to get raw data: %w", err)
	}
	req := storage.CreateQuestRequest{}
	if err := json.Unmarshal(data, &req); err != nil {
		return xerrors.Errorf("failed to unmarshall request: %w", err)
	}
	if err := validate.ImageURL(c, h.fetcher, req.MediaLink); err != nil {
		return aerrors.WrapHTTP(http.StatusUnsupportedMediaType, err)
	}
	uauth, err := jwt.GetUserFromContext(c)
	if err != nil {
		return aerrors.WrapHTTP(http.StatusUnauthorized, err)
	}
	req.Creator = uauth

	s, err := h.clientFactory.NewStorage(c, dbnode.Master)
	if err != nil {
		return xerrors.Errorf("failed to get storage: %w", err)
	}
	quest, err := s.CreateQuest(c, &req)
	if err != nil {
		return xerrors.Errorf("failed to create quest: %w", err)
	}
	c.JSON(http.StatusOK, quest)

	logging.Info(c, "created quest",
		zap.String("quest_id", quest.ID),
		zap.String("quest_name", quest.Name),
		zap.String("creator_id", uauth.ID),
		zap.String("creator_name", uauth.Username),
	)
	return nil
}

// HandleGet handles GET /quest/:id request
//
//	@Summary	Get quest by id
//	@Param		quest_id	path		string	true	"Quest ID"
//	@Success	200			{object}	storage.Quest
//	@Failure	404
//	@Router		/quest/{quest_id} [get]
func (h *Handler) HandleGet(c *gin.Context) error {
	questId := c.Param("id")
	req := storage.GetQuestRequest{ID: questId}
	s, err := h.clientFactory.NewStorage(c, dbnode.Alive)
	if err != nil {
		return xerrors.Errorf("failed to get storage: %w", err)
	}
	user, err := s.GetQuest(c, &req)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return aerrors.NewHttpError(http.StatusNotFound, "quest with id %q not found", req.ID)
		}
		return xerrors.Errorf("failed to get quest: %w", err)
	}
	c.JSON(http.StatusOK, user)
	return nil
}

// HandleUpdate handles POST /quest/:id request
//
//		@Summary	Update main quest information
//		@Param		quest_id	path		string						true	"Quest ID"
//		@Param		request		body		storage.UpdateQuestRequest	true	"Quest information to update"
//		@Success	200			{object}	storage.Quest
//	    @Failure    401
//	    @Failure    403
//		@Failure	404
//		@Failure	415
//		@Router		/quest/{quest_id} [post]
func (h *Handler) HandleUpdate(c *gin.Context) error {
	data, err := c.GetRawData()
	if err != nil {
		return xerrors.Errorf("failed to get raw data: %w", err)
	}
	req := storage.UpdateQuestRequest{}
	if err := json.Unmarshal(data, &req); err != nil {
		return xerrors.Errorf("failed to unmarshall request: %w", err)
	}
	req.ID = c.Param("id")
	if err := validate.ImageURL(c, h.fetcher, req.MediaLink); err != nil {
		return aerrors.WrapHTTP(http.StatusUnsupportedMediaType, err)
	}
	uauth, err := jwt.GetUserFromContext(c)
	if err != nil {
		return aerrors.WrapHTTP(http.StatusUnauthorized, err)
	}

	s, tx, err := h.clientFactory.NewStorageTx(c, nil)
	if err != nil {
		return xerrors.Errorf("failed to get storage: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	quest, err := s.UpdateQuest(c, &req)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return aerrors.NewHttpError(http.StatusNotFound, "quest with id %q not found", req.ID)
		}
		return xerrors.Errorf("failed to update quest: %w", err)
	}
	if quest.Creator == nil || quest.Creator.Username != uauth.Username {
		return aerrors.NewHttpError(http.StatusForbidden, "only creator can update their quest")
	}
	if err := tx.Commit(); err != nil {
		return xerrors.Errorf("failed to commit: %w", err)
	}

	c.JSON(http.StatusOK, quest)
	return nil
}

// HandleDelete handles DELETE /quest/:id request
//
//		@Summary	Delete quest
//		@Param		quest_id	path		string						true	"Quest ID"
//		@Success	200
//	    @Failure    401
//	    @Failure    403
//	    @Failure    404
//		@Router		/quest/{quest_id} [delete]
func (h *Handler) HandleDelete(c *gin.Context) error {
	id := c.Param("id")
	uauth, err := jwt.GetUserFromContext(c)
	if err != nil {
		return aerrors.WrapHTTP(http.StatusUnauthorized, err)
	}
	s, err := h.clientFactory.NewStorage(c, dbnode.Master)
	if err != nil {
		return xerrors.Errorf("failed to get storage: %w", err)
	}

	q, err := s.GetQuest(c, &storage.GetQuestRequest{ID: id})
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return aerrors.NewHttpError(http.StatusNotFound, "quest not found")
		}
		return xerrors.Errorf("failed to get quest: %w", err)
	}
	if q.Creator == nil || q.Creator.ID != uauth.ID {
		return aerrors.NewHttpError(http.StatusForbidden, "cannot delete others' quests")
	}

	if err := s.DeleteQuest(c, &storage.DeleteQuestRequest{ID: id}); err != nil {
		return xerrors.Errorf("%w", err)
	}
	return nil
}
