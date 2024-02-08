package auth

import (
	"encoding/json"
	"errors"
	"net/http"

	"google.golang.org/api/idtoken"

	"go.uber.org/zap"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"golang.org/x/xerrors"

	"questspace/internal/hasher"
	"questspace/internal/validate"
	aerrors "questspace/pkg/application/errors"
	"questspace/pkg/application/logging"
	"questspace/pkg/auth/jwt"
	"questspace/pkg/storage"
)

const defaultAvatarURLTmpl = "https://api.dicebear.com/7.x/thumbs/svg?seed="

type Handler struct {
	storage        storage.UserStorage
	fetcher        http.Client
	pwHasher       hasher.Hasher
	tokenGenerator jwt.Parser
	externalClient idtoken.Validator
}

func (h *Handler) HandleBasic(c *gin.Context) error {
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

	if req.Password == "" {
		return aerrors.NewHttpError(http.StatusBadRequest, "unexpected empty password")
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
	token, err := h.tokenGenerator.CreateToken(user)
	if err != nil {
		// TODO(svayp11): make this code work and uncomment
		//if deleteErr := h.storage.DeleteUser(c, &storage.DeleteUserRequest{ID: user.ID}); err != nil {
		//	return xerrors.Errorf("failed to rollback user after failed token issue: %w", deleteErr)
		//}
		return xerrors.Errorf("failed to issue token: %w", err)
	}

	// TODO(svayp11): set http-only
	c.SetCookie("access_token", token, 60*60, "/", "questspace.app", true, false)
	c.JSON(http.StatusOK, user)

	logging.Info(c, "basic registration done",
		zap.String("username", user.Username),
		zap.String("user_id", user.Id),
	)
	return nil
}

type googleOAuthRequest struct {
	idToken string `json:"id_token"`
}

func (h *Handler) HandleGoogle(c *gin.Context) error {
	data, err := c.GetRawData()
	if err != nil {
		return xerrors.Errorf("failed to get raw data: %w", err)
	}
	req := googleOAuthRequest{}
	if err := json.Unmarshal(data, &req); err != nil {
		return xerrors.Errorf("failed to unmarshall request: %w", err)
	}

	// TODO(svayp11): INSERT CLIENT_ID
	pld, err := h.externalClient.Validate(c, req.idToken, "CLIENT_ID")
	if err != nil {
		return aerrors.NewHttpError(http.StatusBadRequest, "invalid google token: %w", err)
	}

	createUserReq := storage.CreateUserRequest{
		Username:  pld.Claims["name"].(string),
		AvatarURL: pld.Claims["picture"].(string),
	}
	user, err := h.storage.CreateUser(c, &createUserReq)
	if err != nil {
		if errors.Is(err, storage.ErrExists) {
			// TODO(svayp11): issue new token
			return aerrors.NewHttpError(http.StatusNotImplemented, "yet to login user")
		}
		return xerrors.Errorf("failed to create user: %w", err)
	}

	token, err := h.tokenGenerator.CreateToken(user)
	if err != nil {
		// TODO(svayp11): make this code work and uncomment
		//if deleteErr := h.storage.DeleteUser(c, &storage.DeleteUserRequest{ID: user.ID}); err != nil {
		//	return xerrors.Errorf("failed to rollback user after failed token issue: %w", deleteErr)
		//}
		return xerrors.Errorf("failed to issue token: %w", err)
	}

	// TODO(svayp11): set http-only
	c.SetCookie("access_token", token, 60*60, "/", "questspace.app", true, false)
	c.JSON(http.StatusOK, user)

	logging.Info(c, "google registration done",
		zap.String("username", user.Username),
		zap.String("user_id", user.Id),
	)
	return nil
}
