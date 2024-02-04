package user

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"golang.org/x/xerrors"

	"questspace/internal/hasher"
	"questspace/internal/validate"
	aerrors "questspace/pkg/application/errors"
	"questspace/pkg/storage"
)

const defaultAvatarURLTmpl = "https://api.dicebear.com/7.x/thumbs/svg?seed="

type CreateHandler struct {
	storage  storage.UserStorage
	fetcher  http.Client
	pwHasher hasher.Hasher
}

func NewCreateHandler(s storage.UserStorage, f http.Client, h hasher.Hasher) CreateHandler {
	return CreateHandler{
		storage:  s,
		fetcher:  f,
		pwHasher: h,
	}
}

// Handle handles POST /user request
//
// @Summary Create user
// @Param request body storage.CreateUserRequest true "Create user request"
// @Success 200 {object} storage.User
// @Failure 400
// @Failure 415
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
	if err := validate.ImageURL(c, h.fetcher, req.AvatarURL); err != nil {
		return aerrors.WrapHTTP(http.StatusUnsupportedMediaType, err)
	}
	if req.AvatarURL == "" {
		seed, err := uuid.NewV4()
		if err != nil {
			return xerrors.Errorf("failed to generate avatar rand seed: %w", err)
		}
		req.AvatarURL = defaultAvatarURLTmpl + seed.String()
	}

	req.Password, err = h.pwHasher.HashString(req.Password)
	if err != nil {
		return xerrors.Errorf("failed to calculate password hash: %w", err)
	}
	user, err := h.storage.CreateUser(c, &req)
	if err != nil {
		if errors.Is(err, storage.ErrExists) {
			return aerrors.NewHttpError(http.StatusBadRequest, "user %q already exits", req.Username)
		}
		return xerrors.Errorf("failed to create user: %w", err)
	}

	c.JSON(http.StatusOK, user)
	return nil
}

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
			return aerrors.NewHttpError(http.StatusNotFound, "user with id %q not found", req.Id)
		}
		return xerrors.Errorf("failed to get user: %w", err)
	}

	c.JSON(http.StatusOK, user)
	return nil
}

type UpdateHandler struct {
	storage  storage.UserStorage
	fetcher  http.Client
	pwHasher hasher.Hasher
}

func NewUpdateHandler(s storage.UserStorage, f http.Client, h hasher.Hasher) UpdateHandler {
	return UpdateHandler{
		storage:  s,
		fetcher:  f,
		pwHasher: h,
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
	if req.OldPassword == "" {
		return aerrors.NewHttpError(http.StatusUnauthorized, "old_password was not provided")
	}
	req.Id = c.Param("id")

	if req.AvatarURL != "" {
		if err := validate.ImageURL(c, h.fetcher, req.AvatarURL); err != nil {
			return aerrors.WrapHTTP(http.StatusUnsupportedMediaType, err)
		}
	}
	userPwHash, err := h.storage.GetUserPasswordHash(c, &storage.GetUserRequest{Id: req.Id, Username: req.Username})
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return aerrors.NewHttpError(http.StatusNotFound, "user with id %q not found", req.Id)
		}
		return xerrors.Errorf("failed to get old user password: %w", err)
	}

	if !h.pwHasher.HasSameHash(req.OldPassword, userPwHash) {
		return aerrors.NewHttpError(http.StatusUnauthorized, "invalid old_password provided")
	}

	user, err := h.storage.UpdateUser(c, &req)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return aerrors.NewHttpError(http.StatusNotFound, "user with id %q not found", req.Id)
		}
		return xerrors.Errorf("failed to update user: %w", err)
	}

	c.JSON(http.StatusOK, user)
	return nil
}
