package quest

import (
	"encoding/json"
	"errors"
	"hash"
	"net/http"

	aerrors "questspace/pkg/application/errors"
	"questspace/pkg/storage"

	"github.com/gin-gonic/gin"
	"golang.org/x/xerrors"
)

type UpdateHandler struct {
	storage storage.QuestStorage
	fetcher http.Client
	hasher  hash.Hash
}

func NewUpdateHandler(s storage.QuestStorage, f http.Client, h hash.Hash) UpdateHandler {
	return UpdateHandler{
		storage: s,
		fetcher: f,
		hasher:  h,
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
		return xerrors.Errorf("failed to ")
	}
	req := storage.UpdateQuestRequest{}
	if err := json.Unmarshal(data, &req); err != nil {
		return xerrors.Errorf("failed to unmarshall request: %w", err)
	}
	req.Id = c.Param("id")

	quest, err := h.storage.UpdateQuest(c, &req)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return aerrors.ErrNotFound
		}
		return xerrors.Errorf("failed to update quest: %w", err)
	}
	c.JSON(http.StatusOK, quest)
	return nil
}
