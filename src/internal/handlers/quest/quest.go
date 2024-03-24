package quest

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/xerrors"

	"questspace/internal/handlers/transport"
	pgdb "questspace/internal/pgdb/client"
	"questspace/internal/questspace/quests"
	"questspace/internal/validate"
	"questspace/pkg/application/httperrors"
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
	req, err := transport.UnmarshalRequestData[storage.CreateQuestRequest](c.Request)
	if err != nil {
		return xerrors.Errorf("%w", err)
	}
	if err := validate.ImageURL(c, h.fetcher, req.MediaLink); err != nil {
		return xerrors.Errorf("%w", err)
	}
	uauth, err := jwt.GetUserFromContext(c)
	if err != nil {
		return xerrors.Errorf("%w", err)
	}
	req.Creator = uauth

	s, err := h.clientFactory.NewStorage(c, dbnode.Master)
	if err != nil {
		return xerrors.Errorf("get storage client: %w", err)
	}
	quest, err := s.CreateQuest(c, req)
	if err != nil {
		return xerrors.Errorf("create quest: %w", err)
	}
	quests.SetStatus(quest)
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
		return xerrors.Errorf("get storage client: %w", err)
	}
	quest, err := s.GetQuest(c, &req)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return httperrors.Errorf(http.StatusNotFound, "not found quest with id %q", req.ID)
		}
		return xerrors.Errorf("get quest: %w", err)
	}
	quests.SetStatus(quest)
	c.JSON(http.StatusOK, quest)
	return nil
}

const defaultPageSize = 50

// HandleGetMany handles GET /quest request
//
//		@Summary	Get many quests sorted by start time and finished status
//	 @Param      fields      query       []string   false  "Fields to return"  Enums(all, registered, owned) minlength(0) maxlength(3)
//
// @Param   page_size     query     int        false  "Number of quests to return for each field" default(50)
// @Param   page_id  query     string     false  "Page ID to return. Mutually exclusive to multiple fields"
// @Success	200			{object}	quests.Quests
// @Failure	400
// @Failure	401
// @Router		/quest [get]
func (h *Handler) HandleGetMany(c *gin.Context) error {
	uauth, err := jwt.GetUserFromContext(c)
	if err != nil {
		return xerrors.Errorf("%w", err)
	}

	fields := c.QueryArray("fields")
	pageSizeStr := c.Query("page_size")
	pageSize := defaultPageSize
	if pageSizeStr != "" {
		var err error
		pageSize, err = strconv.Atoi(pageSizeStr)
		if err != nil {
			return httperrors.Errorf(http.StatusBadRequest, "parse page size: %w", err)
		}
	}
	pageID := c.Query("page_id")

	s, err := h.clientFactory.NewStorage(c, dbnode.Alive)
	if err != nil {
		return xerrors.Errorf("get storage: %w", err)
	}
	gotQuests, err := quests.ReadQuests(c, s, uauth, fields, pageID, pageSize)
	if err != nil {
		return xerrors.Errorf("read quests: %w", err)
	}

	c.JSON(http.StatusOK, gotQuests)
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
	req, err := transport.UnmarshalRequestData[storage.UpdateQuestRequest](c.Request)
	if err != nil {
		return xerrors.Errorf("%w", err)
	}
	req.ID = c.Param("id")
	if err := validate.ImageURL(c, h.fetcher, req.MediaLink); err != nil {
		return xerrors.Errorf("%w", err)
	}
	uauth, err := jwt.GetUserFromContext(c)
	if err != nil {
		return xerrors.Errorf("%w", err)
	}

	s, tx, err := h.clientFactory.NewStorageTx(c, nil)
	if err != nil {
		return xerrors.Errorf("get storage: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	quest, err := s.UpdateQuest(c, req)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return httperrors.Errorf(http.StatusNotFound, "not found quest with id %q", req.ID)
		}
		return xerrors.Errorf("failed to update quest: %w", err)
	}
	if quest.Creator == nil || quest.Creator.Username != uauth.Username {
		return httperrors.New(http.StatusForbidden, "only creator can update their quest")
	}
	if err := tx.Commit(); err != nil {
		return xerrors.Errorf("commit transaction: %w", err)
	}
	quests.SetStatus(quest)
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
		return xerrors.Errorf("%w", err)
	}
	s, err := h.clientFactory.NewStorage(c, dbnode.Master)
	if err != nil {
		return xerrors.Errorf("get storage client: %w", err)
	}

	q, err := s.GetQuest(c, &storage.GetQuestRequest{ID: id})
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return httperrors.Errorf(http.StatusNotFound, "quest not found")
		}
		return xerrors.Errorf("get quest: %w", err)
	}
	if q.Creator == nil || q.Creator.ID != uauth.ID {
		return httperrors.New(http.StatusForbidden, "cannot delete others' quests")
	}

	if err := s.DeleteQuest(c, &storage.DeleteQuestRequest{ID: id}); err != nil {
		return xerrors.Errorf("%w", err)
	}
	return nil
}
