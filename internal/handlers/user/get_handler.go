package user

import (
	"errors"
	"net/http"
	aerrors "questspace/pkg/application/errors"
	"questspace/pkg/storage"

	"github.com/gin-gonic/gin"
	"golang.org/x/xerrors"
)

type GetHandler struct {
	storage storage.UserStorage
}

func NewGetHandler(s storage.UserStorage) GetHandler {
	return GetHandler{
		storage: s,
	}
}

// Handle handles GET /user/:id request
//
// @Summary Get user by id
// @Param user_id path string true "User ID"
// @Success 200 {object} storage.User
// @Failure 404
// @Router /user/{user_id} [get]
func (h GetHandler) Handle(c *gin.Context) error {
	userId := c.Param("id")
	req := &storage.GetUserRequest{Id: userId}
	user, err := h.storage.GetUser(c, req)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return aerrors.ErrNotFound
		}
		return xerrors.Errorf("failed to get user: %w", err)
	}

	c.JSON(http.StatusOK, user)
	return nil
}
