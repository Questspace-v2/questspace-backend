package user

import (
	"database/sql"
	"errors"
	"net/http"

	"questspace/internal/handlers/auth"
	"questspace/internal/handlers/transport"

	"github.com/gin-gonic/gin"
	"golang.org/x/xerrors"

	"questspace/internal/hasher"
	pgdb "questspace/internal/pgdb/client"
	"questspace/internal/validate"
	"questspace/pkg/application/httperrors"
	"questspace/pkg/auth/jwt"
	"questspace/pkg/dbnode"
	"questspace/pkg/storage"
)

type GetHandler struct {
	clientFactory pgdb.QuestspaceClientFactory
}

func NewGetHandler(cf pgdb.QuestspaceClientFactory) *GetHandler {
	return &GetHandler{
		clientFactory: cf,
	}
}

// Handle handles GET /user/:id request
//
// @Summary	Get user by id
// @Tags	Users
// @Param	user_id	path		string	true	"User ID"
// @Success	200		{object}	storage.User
// @Failure	404
// @Router	/user/{user_id} [get]
func (h *GetHandler) Handle(c *gin.Context) error {
	userId := c.Param("id")
	req := &storage.GetUserRequest{ID: userId}
	s, err := h.clientFactory.NewStorage(c, dbnode.Alive)
	if err != nil {
		return xerrors.Errorf("get storage: %w", err)
	}
	user, err := s.GetUser(c, req)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return httperrors.Errorf(http.StatusNotFound, "user with id %q not found", req.ID)
		}
		return xerrors.Errorf("get user: %w", err)
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

func NewUpdateHandler(cf pgdb.QuestspaceClientFactory, f http.Client, h hasher.Hasher, g jwt.Parser) *UpdateHandler {
	return &UpdateHandler{
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
// @Summary		Update user public data such as username or avatar and returns new auth data
// @Tags		Users
// @Param		user_id	path	string							true	"User ID"
// @Param		request	body	user.UpdatePublicDataRequest	true	"Public data to set for user"
// @Success		200	{object}	auth.Response
// @Failure		401
// @Failure		403
// @Failure		404
// @Failure		422
// @Router		/user/{user_id} [post]
// @Security 	ApiKeyAuth
func (h *UpdateHandler) HandleUser(c *gin.Context) error {
	req, err := transport.UnmarshalRequestData[UpdatePublicDataRequest](c.Request)
	if err != nil {
		return xerrors.Errorf("%w", err)
	}
	uauth, err := jwt.GetUserFromContext(c)
	if err != nil {
		return xerrors.Errorf("%w", err)
	}
	id := c.Param("id")
	if uauth.ID != id {
		return httperrors.Errorf(http.StatusForbidden, "cannot change data of another user")
	}
	if err := validate.ImageURL(c, h.fetcher, req.AvatarURL); err != nil {
		return httperrors.WrapWithCode(http.StatusUnsupportedMediaType, err)
	}

	s, tx, err := h.clientFactory.NewStorageTx(c, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return xerrors.Errorf("get storage: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	user, err := s.UpdateUser(c, &storage.UpdateUserRequest{ID: id, Username: req.Username, AvatarURL: req.AvatarURL})
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return httperrors.Errorf(http.StatusNotFound, "user with id %q not found", id)
		}
		if errors.Is(err, storage.ErrExists) {
			return httperrors.New(http.StatusBadRequest, "user with such name already exists")
		}
		return xerrors.Errorf("update user: %w", err)
	}
	token, err := h.tokenGenerator.CreateToken(user)
	if err != nil {
		return xerrors.Errorf("issue token: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return xerrors.Errorf("commit tx: %w", err)
	}

	resp := auth.Response{
		User:        user,
		AccessToken: token,
	}
	c.JSON(http.StatusOK, &resp)
	return nil
}

type UpdatePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// HandlePassword handles POST /user/:id/password request
//
// @Summary		Update user password
// @Tags		Users
// @Param		user_id	path	string						true	"User ID"
// @Param		request	body	user.UpdatePasswordRequest	true	"Old and new password"
// @Success		200	{object}	storage.User
// @Failure		401
// @Failure		403
// @Route		/user/{user_id}/password [post]
// @Security 	ApiKeyAuth
func (h *UpdateHandler) HandlePassword(c *gin.Context) error {
	req, err := transport.UnmarshalRequestData[UpdatePasswordRequest](c.Request)
	if err != nil {
		return xerrors.Errorf("%w", err)
	}
	uauth, err := jwt.GetUserFromContext(c)
	if err != nil {
		return httperrors.WrapWithCode(http.StatusUnauthorized, err)
	}
	id := c.Param("id")
	if uauth.ID != id {
		return httperrors.Errorf(http.StatusForbidden, "cannot change data of another user")
	}

	s, err := h.clientFactory.NewStorage(c, dbnode.Master)
	if err != nil {
		return xerrors.Errorf("get storage: %w", err)
	}
	oldPw, err := s.GetUserPasswordHash(c, &storage.GetUserRequest{ID: id})
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return httperrors.Errorf(http.StatusNotFound, "user with id %q not found", id)
		}
		return xerrors.Errorf("lookup user password: %w", err)
	}
	if !h.pwHasher.HasSameHash(req.OldPassword, oldPw) {
		return httperrors.Errorf(http.StatusForbidden, "invalid password")
	}
	pwHash, err := h.pwHasher.HashString(req.NewPassword)
	if err != nil {
		return xerrors.Errorf("hash password: %w", err)
	}
	user, err := s.UpdateUser(c, &storage.UpdateUserRequest{ID: id, Password: pwHash})
	if err != nil {
		return xerrors.Errorf("update user: %w", err)
	}

	c.JSON(http.StatusOK, user)
	return nil
}

// HandleDelete handles DELETE /user/:id request
//
// @Summary		Delete user account
// @Tags		Users
// @Param		user_id	path	string	true	"User ID"
// @Success		200
// @Failure		401
// @Failure		403
// @Failure		404
// @Router		/user/{user_id} [delete]
// @Security 	ApiKeyAuth
func (h *UpdateHandler) HandleDelete(c *gin.Context) error {
	id := c.Param("id")
	uauth, err := jwt.GetUserFromContext(c)
	if err != nil {
		return httperrors.WrapWithCode(http.StatusUnauthorized, err)
	}
	if uauth.ID != id {
		return httperrors.Errorf(http.StatusForbidden, "cannot delete other users")
	}

	req := storage.DeleteUserRequest{ID: id}
	s, err := h.clientFactory.NewStorage(c, dbnode.Master)
	if err != nil {
		return xerrors.Errorf("get storage: %w", err)
	}
	if err := s.DeleteUser(c, &req); err != nil {
		return xerrors.Errorf("delete %q: %w", uauth.Username, err)
	}
	return nil
}
