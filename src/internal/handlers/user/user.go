package user

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/xerrors"

	"questspace/internal/hasher"
	pgdb "questspace/internal/pgdb/client"
	"questspace/internal/validate"
	aerrors "questspace/pkg/application/errors"
	"questspace/pkg/auth/jwt"
	"questspace/pkg/dbnode"
	"questspace/pkg/storage"
)

type GetHandler struct {
	clientFactory pgdb.QuestspaceClientFactory
}

func NewGetHandler(cf pgdb.QuestspaceClientFactory) GetHandler {
	return GetHandler{
		clientFactory: cf,
	}
}

// Handle handles GET /user/:id request
//
//	@Summary	Get user by id
//	@Param		user_id	path		string	true	"User ID"
//	@Success	200		{object}	storage.User
//	@Failure	404
//	@Router		/user/{user_id} [get]
func (h GetHandler) Handle(c *gin.Context) error {
	userId := c.Param("id")
	req := &storage.GetUserRequest{ID: userId}
	s, err := h.clientFactory.NewStorage(c, dbnode.Alive)
	if err != nil {
		return xerrors.Errorf("failed to get storage: %w", err)
	}
	user, err := s.GetUser(c, req)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return aerrors.NewHttpError(http.StatusNotFound, "user with id %q not found", req.ID)
		}
		return xerrors.Errorf("failed to get user: %w", err)
	}

	c.JSON(http.StatusOK, user)
	return nil
}

type UpdateHandler struct {
	clientFactory  pgdb.QuestspaceClientFactory
	fetcher        http.Client
	pwHasher       hasher.Hasher
	tokenGenerator jwt.Parser
}

func NewUpdateHandler(cf pgdb.QuestspaceClientFactory, f http.Client, h hasher.Hasher, g jwt.Parser) UpdateHandler {
	return UpdateHandler{
		clientFactory:  cf,
		fetcher:        f,
		pwHasher:       h,
		tokenGenerator: g,
	}
}

type UpdatePublicDataRequest struct {
	Username  string `json:"username"`
	AvatarURL string `json:"avatar_url"`
}

// HandleUser handles POST /user/:id request
//
//	@Summary								Update user public data such as username or avatar and returns new auth data
//	@Param									user_id	path	string							true	"User ID"
//	@Param									request	body	user.UpdatePublicDataRequest	true	"Public data to set for user"
//	@securitydefinitions.oauth2.application	JWT user token
//	@in										header
//	@name									Authorization
//	@Success								200	{object}	storage.User
//	@Failure								401
//	@Failure								404
//	@Failure								422
//	@Router									/user/{user_id} [post]
func (h UpdateHandler) HandleUser(c *gin.Context) error {
	data, err := c.GetRawData()
	if err != nil {
		return xerrors.Errorf("failed to ")
	}
	req := UpdatePublicDataRequest{}
	if err := json.Unmarshal(data, &req); err != nil {
		return xerrors.Errorf("failed to unmarshall request: %w", err)
	}
	uauth, err := jwt.GetUserFromContext(c)
	if err != nil {
		return err
	}
	id := c.Param("id")
	if uauth.ID != id {
		return aerrors.NewHttpError(http.StatusForbidden, "cannot change data of another user")
	}
	if req.AvatarURL != "" {
		if err := validate.ImageURL(c, h.fetcher, req.AvatarURL); err != nil {
			return aerrors.WrapHTTP(http.StatusUnsupportedMediaType, err)
		}
	}

	s, tx, err := h.clientFactory.NewStorageTx(c, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return xerrors.Errorf("failed to get storage: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	user, err := s.UpdateUser(c, &storage.UpdateUserRequest{ID: id, Username: req.Username, AvatarURL: req.AvatarURL})
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return aerrors.NewHttpError(http.StatusNotFound, "user with id %q not found", id)
		}
		return xerrors.Errorf("failed to update user: %w", err)
	}
	token, err := h.tokenGenerator.CreateToken(user)
	if err != nil {
		return xerrors.Errorf("failed to issue token: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return xerrors.Errorf("failed to commit transaction: %w", err)
	}

	c.SetCookie("access_token", token, 60*60, "/", "questspace.app", true, false)
	c.JSON(http.StatusOK, user)
	return nil
}

type UpdatePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// HandlePassword handles POST /user/:id/password request
//
//	@Summary								Update user password
//	@Param									user_id	path	string						true	"User ID"
//	@Param									request	body	user.UpdatePasswordRequest	true	"Old and new password"
//	@securitydefinitions.oauth2.application	JWT user token
//	@in										header
//	@name									Authorization
//	@Success								200	{object}	storage.User
//	@Failure								401
//	@Failure								403
//	@Failure								404
//	@Router									/user/{user_id}/password [post]
func (h UpdateHandler) HandlePassword(c *gin.Context) error {
	data, err := c.GetRawData()
	if err != nil {
		return xerrors.Errorf("failed to ")
	}
	req := UpdatePasswordRequest{}
	if err := json.Unmarshal(data, &req); err != nil {
		return xerrors.Errorf("failed to unmarshall request: %w", err)
	}
	uauth, err := jwt.GetUserFromContext(c)
	if err != nil {
		return err
	}
	id := c.Param("id")
	if uauth.ID != id {
		return aerrors.NewHttpError(http.StatusForbidden, "cannot change data of another user")
	}

	s, err := h.clientFactory.NewStorage(c, dbnode.Master)
	if err != nil {
		return xerrors.Errorf("failed to get storage: %w", err)
	}
	oldPw, err := s.GetUserPasswordHash(c, &storage.GetUserRequest{ID: id})
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return aerrors.NewHttpError(http.StatusNotFound, "user with id %q not found", id)
		}
		return xerrors.Errorf("failed to lookup user password: %w", err)
	}
	if !h.pwHasher.HasSameHash(oldPw, req.OldPassword) {
		return aerrors.NewHttpError(http.StatusForbidden, "invalid password")
	}

	user, err := s.UpdateUser(c, &storage.UpdateUserRequest{ID: id, Password: req.NewPassword})
	if err != nil {
		return xerrors.Errorf("failed to update user: %w", err)
	}

	c.JSON(http.StatusOK, user)
	return nil
}
