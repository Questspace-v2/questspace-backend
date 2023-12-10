package quest

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/xerrors"

	aerrors "questspace/pkg/application/errors"
	"questspace/pkg/storage"
)

type CreateHandler struct {
	storage storage.QuestStorage
}

func NewCreateHandler(s storage.QuestStorage) CreateHandler {
	return CreateHandler{
		storage: s,
	}
}

// Handle handles POST /quest request
//
// @Summary Create quest
// @Param request body storage.CreateQuestRequest true "Create quest request"
// @Success 200 {object} storage.Quest
// @Failure 400
// @Failure 422
// @Router /quest [post]
func (h CreateHandler) Handle(c *gin.Context) error {
	data, err := c.GetRawData()
	if err != nil {
		return xerrors.Errorf("failed to get raw data: %w", err)
	}
	req := storage.CreateQuestRequest{}
	if err := json.Unmarshal(data, &req); err != nil {
		return xerrors.Errorf("failed to unmarshall request: %w", err)
	}

	quest, err := h.storage.CreateQuest(c, &req)
	if err != nil {
		return xerrors.Errorf("failed to create quest: %w", err)
	}
	c.JSON(http.StatusOK, quest)
	return nil
}

type GetHandler struct {
	storage storage.QuestStorage
}

func NewGetHandler(s storage.QuestStorage) GetHandler {
	return GetHandler{
		storage: s,
	}
}

// Handle handles GET /quest/:id request
//
// @Summary Get quest by id
// @Param quest_id path string true "Quest ID"
// @Success 200 {object} storage.Quest
// @Failure 404
// @Router /quest/{quest_id} [get]
func (h GetHandler) Handle(c *gin.Context) error {
	questId := c.Param("id")
	req := &storage.GetQuestRequest{Id: questId}
	user, err := h.storage.GetQuest(c, req)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return aerrors.NewHttpError(http.StatusNotFound, "quest with id %q not found", req.Id)
		}
		return xerrors.Errorf("failed to get quest: %w", err)
	}
	c.JSON(http.StatusOK, user)
	return nil
}

type UpdateHandler struct {
	storage storage.QuestStorage
}

func NewUpdateHandler(s storage.QuestStorage) UpdateHandler {
	return UpdateHandler{
		storage: s,
	}
}

// Handle handles POST /quest/:id request
//
// @Summary Update quest
// @Param quest_id path string true "Quest ID"
// @Param request body storage.UpdateQuestRequest true "Update quest request"
// @Success 200 {object} storage.Quest
// @Failure 404
// @Failure 422
// @Router /quest/{quest_id} [post]
func (h UpdateHandler) Handle(c *gin.Context) error {
	data, err := c.GetRawData()
	if err != nil {
		return xerrors.Errorf("failed to get raw data: %w", err)
	}
	req := storage.UpdateQuestRequest{}
	if err := json.Unmarshal(data, &req); err != nil {
		return xerrors.Errorf("failed to unmarshall request: %w", err)
	}
	req.Id = c.Param("id")

	quest, err := h.storage.UpdateQuest(c, &req)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return aerrors.NewHttpError(http.StatusNotFound, "quest with id %q not found", req.Id)
		}
		return xerrors.Errorf("failed to update quest: %w", err)
	}
	c.JSON(http.StatusOK, quest)
	return nil
}
