package user

import (
	"encoding/json"
	"errors"
	"hash"
	"net/http"
	"questspace/internal/validate"
	aerrors "questspace/pkg/application/errors"
	"questspace/pkg/storage"

	"golang.org/x/xerrors"

	"github.com/gin-gonic/gin"
)

type UpdateHandler struct {
	storage storage.UserStorage
	fetcher http.Client
	hasher  hash.Hash
}

func NewUpdateHandler(s storage.UserStorage, f http.Client, h hash.Hash) UpdateHandler {
	return UpdateHandler{
		storage: s,
		fetcher: f,
		hasher:  h,
	}
}

// Handle handles POST /user/:id request
//
// @Summary Update user
// @Param user_id path string true "User ID"
// @Param request body storage.UpdateUserRequest true "Update user request"
// @Success 200 {object} storage.User
// @Failure 404
// @Failure 422
// @Router /user/{user_id} [post]
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
	h.hasher.Write([]byte(req.Password))
	req.Password = string(h.hasher.Sum(nil))
	h.hasher.Reset()
	user, err := h.storage.UpdateUser(c, &req)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return aerrors.ErrNotFound
		}
		return xerrors.Errorf("failed to update user: %w", err)
	}

	c.JSON(http.StatusOK, user)

	return nil
}
