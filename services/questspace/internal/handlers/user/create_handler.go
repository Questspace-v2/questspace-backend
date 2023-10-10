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

const defaultAvatarURL = "https://api.dicebear.com/7.x/thumbs/svg"

type CreateHandler struct {
	storage storage.UserStorage
	fetcher http.Client
}

func NewCreateHandler(s storage.UserStorage, f http.Client) CreateHandler {
	return CreateHandler{
		storage: s,
		fetcher: f,
	}
}

func (h CreateHandler) Handle(c *gin.Context) error {
	data, err := c.GetRawData()
	if err != nil {
		return xerrors.Errorf("failed to get raw data: %w", err)
	}
	req := storage.CreateUserRequest{}
	if err := json.Unmarshal(data, &req); err != nil {
		return xerrors.Errorf("failed to unmarshall request: %w", err)
	}
	if err := validate.ImageURL(h.fetcher, req.AvatarURL); err != nil {
		return xerrors.Errorf("failed to validate image: %w", err)
	}
	if req.AvatarURL == "" {
		req.AvatarURL = defaultAvatarURL
	}
	user, err := h.storage.CreateUser(c, &req)
	if err != nil {
		if errors.Is(err, storage.ErrExists) {
			return aerrors.ErrBadRequest
		}
		return xerrors.Errorf("failed to create user: %w", err)
	}
	c.JSON(http.StatusOK, user)
	return nil
}
