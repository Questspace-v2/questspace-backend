package user

import (
	"encoding/json"
	"errors"
	"net/http"
	aerrors "questspace/pkg/application/errors"
	"questspace/pkg/storage"
	"questspace/services/questspace/internal/validate"

	"golang.org/x/xerrors"

	"github.com/gin-gonic/gin"
)

type UpdateHandler struct {
	storage storage.UserStorage
	fetcher http.Client
}

func NewUpdateHandler(s storage.UserStorage, f http.Client) UpdateHandler {
	return UpdateHandler{
		storage: s,
		fetcher: f,
	}
}

func (h UpdateHandler) Handle(c *gin.Context) error {
	data, err := c.GetRawData()
	if err != nil {
		return xerrors.Errorf("failed to ")
	}
	req := storage.UpdateUserRequest{}
	if err := json.Unmarshal(data, &req); err != nil {
		return xerrors.Errorf("failed to unmarshall request: %w", err)
	}
	req.Id = c.Param("id")

	if req.AvatarURL != "" {
		if err := validate.ImageURL(h.fetcher, req.AvatarURL); err != nil {
			return xerrors.Errorf("failed to validate an image: %w", err)
		}
	}

	user, err := h.storage.UpdateUser(c, &req)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return aerrors.ErrNotFound
		}
		return xerrors.Errorf("failed to update user: %w", err)
	}

	// Set password to "" in case it is not empty
	user.Password = ""
	c.JSON(http.StatusOK, user)

	return nil
}
