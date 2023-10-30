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

const defaultAvatarURL = "https://api.dicebear.com/7.x/thumbs/svg"

type CreateHandler struct {
	storage storage.UserStorage
	fetcher http.Client
	hasher  hash.Hash
}

func NewCreateHandler(s storage.UserStorage, f http.Client, h hash.Hash) CreateHandler {
	return CreateHandler{
		storage: s,
		fetcher: f,
		hasher:  h,
	}
}

// Handle handles POST /user request
//
// @Summary Create user
// @Param request body storage.CreateUserRequest true "Create user request"
// @Success 200 {object} storage.User
// @Failure 400
// @Failure 422
// @Router /user [post]
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
	h.hasher.Write([]byte(req.Password))
	req.Password = string(h.hasher.Sum(nil))
	h.hasher.Reset()
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
