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

func (h GetHandler) HandleQS(c *gin.Context) error {
	userId := c.Query("id")
	userName := c.Query("username")
	if userId == "" && userName == "" {
		return xerrors.Errorf("either id or username must be set: %w", aerrors.ErrBadRequest)
	}
	req := &storage.GetUserRequest{Id: userId, Username: userName}
	return h.handle(c, req)
}

func (h GetHandler) HandlePath(c *gin.Context) error {
	userId := c.Param("id")
	req := &storage.GetUserRequest{Id: userId}
	return h.handle(c, req)
}

func (h GetHandler) handle(c *gin.Context, req *storage.GetUserRequest) error {
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
