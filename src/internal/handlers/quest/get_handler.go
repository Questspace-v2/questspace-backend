package quest

import (
	"errors"
	"net/http"

	aerrors "questspace/pkg/application/errors"
	"questspace/pkg/storage"

	"github.com/gin-gonic/gin"
	"golang.org/x/xerrors"
)

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
			return aerrors.ErrNotFound
		}
		return xerrors.Errorf("failed to get quest: %w", err)
	}
	c.JSON(http.StatusOK, user)
	return nil
}
