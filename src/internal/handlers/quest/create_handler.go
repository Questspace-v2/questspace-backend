package quest

import (
	"encoding/json"
	"hash"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/xerrors"

	"questspace/pkg/storage"
)

type CreateHandler struct {
	storage storage.QuestStorage
	fetcher http.Client
	hasher  hash.Hash
}

func NewCreateHandler(s storage.QuestStorage, f http.Client, h hash.Hash) CreateHandler {
	return CreateHandler{
		storage: s,
		fetcher: f,
		hasher:  h,
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
