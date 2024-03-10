package auth

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/zap"
	"golang.org/x/xerrors"
	"google.golang.org/api/idtoken"

	"questspace/internal/hasher"
	pgdb "questspace/internal/pgdb/client"
	"questspace/internal/validate"
	"questspace/pkg/application/httperrors"
	"questspace/pkg/application/logging"
	"questspace/pkg/auth/jwt"
	"questspace/pkg/dbnode"
	"questspace/pkg/storage"
)

const defaultAvatarURLTmpl = "https://api.dicebear.com/7.x/thumbs/svg?seed="

type Handler struct {
	clientFactory  pgdb.QuestspaceClientFactory
	fetcher        http.Client
	pwHasher       hasher.Hasher
	tokenGenerator jwt.Parser
	externalClient idtoken.Validator
}

func NewHandler(cf pgdb.QuestspaceClientFactory, f http.Client, h hasher.Hasher, tg jwt.Parser) *Handler {
	return &Handler{
		clientFactory:  cf,
		fetcher:        f,
		pwHasher:       h,
		tokenGenerator: tg,
	}
}

// HandleBasicSignUp handles POST /auth/register request
//
//	@Summary	Register new user and return auth data
//	@Param		request	body		storage.CreateUserRequest	true	"Create user request"
//	@Success	200		{object}	storage.User
//	@Failure	400
//	@Failure	415
//	@Router		/auth/register [post]
func (h *Handler) HandleBasicSignUp(c *gin.Context) error {
	data, err := c.GetRawData()
	if err != nil {
		return xerrors.Errorf("get raw data: %w", err)
	}
	req := storage.CreateUserRequest{}
	if err := json.Unmarshal(data, &req); err != nil {
		return xerrors.Errorf("unmarshall request: %w", err)
	}
	if err := validate.ImageURL(c, h.fetcher, req.AvatarURL); err != nil {
		return xerrors.Errorf("%w", err)
	}
	if req.AvatarURL == "" {
		req.AvatarURL = defaultAvatarURLTmpl + uuid.Must(uuid.NewV4()).String()
	}
	if req.Password == "" {
		return httperrors.New(http.StatusBadRequest, "unexpected empty password")
	}
	req.Password, err = h.pwHasher.HashString(req.Password)
	if err != nil {
		return xerrors.Errorf("calculate password hash: %w", err)
	}

	s, tx, err := h.clientFactory.NewStorageTx(c, nil)
	if err != nil {
		return xerrors.Errorf("get storage client: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	user, err := s.CreateUser(c, &req)
	if err != nil {
		if errors.Is(err, storage.ErrExists) {
			return httperrors.Errorf(http.StatusBadRequest, "user %q already exits", req.Username)
		}
		return xerrors.Errorf("create user: %w", err)
	}

	if err := h.sendAuthDataAndCommit(c, user, tx); err != nil {
		return err
	}

	logging.Info(c, "basic registration done",
		zap.String("username", user.Username),
		zap.String("user_id", user.ID),
	)
	return nil
}

type SignInRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// HandleBasicSignIn handles POST /auth/sign-in request
//
//	@Summary	Sign in to user account and return auth data
//	@Param		request	body		auth.SignInRequest	true	"Sign in request"
//	@Success	200		{object}	storage.User
//	@Failure	400
//	@Failure	403
//	@Failure	415
//	@Router		/auth/sign-in [post]
func (h *Handler) HandleBasicSignIn(c *gin.Context) error {
	data, err := c.GetRawData()
	if err != nil {
		return xerrors.Errorf("get raw data: %w", err)
	}
	req := SignInRequest{}
	if err := json.Unmarshal(data, &req); err != nil {
		return xerrors.Errorf("unmarshall request: %w", err)
	}

	s, err := h.clientFactory.NewStorage(c, dbnode.Alive)
	if err != nil {
		return xerrors.Errorf("get storage client: %w", err)
	}
	pwHash, err := s.GetUserPasswordHash(c, &storage.GetUserRequest{Username: req.Username})
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return httperrors.Errorf(http.StatusNotFound, "user %s not exists", req.Username)
		}
		return xerrors.Errorf("lookup user password: %w", err)
	}
	if !h.pwHasher.HasSameHash(req.Password, pwHash) {
		return httperrors.New(http.StatusForbidden, "invalid password")
	}
	user, err := s.GetUser(c, &storage.GetUserRequest{Username: req.Username})
	if err != nil {
		return xerrors.Errorf("get user: %w", err)
	}
	token, err := h.tokenGenerator.CreateToken(user)
	if err != nil {
		return xerrors.Errorf("issue token: %w", err)
	}

	c.SetCookie(jwt.AuthCookieName, token, 60*60, "/", "questspace.app", true, false)
	c.JSON(http.StatusOK, user)
	return nil
}

type GoogleOAuthRequest struct {
	IdToken string `json:"id_token"`
}

// HandleGoogle handles POST /auth/google request
//
//	@Summary	Register new or sign in old user using Google OAuth2.0
//	@Param		request	body		auth.GoogleOAuthRequest	true	"Google OAuth request"
//	@Success	200		{object}	storage.User
//	@Failure	400
//	@Failure	415
//	@Router		/auth/google [post]
//
// TODO(svayp11): WIP endpoint
func (h *Handler) HandleGoogle(c *gin.Context) error {
	data, err := c.GetRawData()
	if err != nil {
		return xerrors.Errorf("get raw data: %w", err)
	}
	req := GoogleOAuthRequest{}
	if err := json.Unmarshal(data, &req); err != nil {
		return xerrors.Errorf("unmarshall request: %w", err)
	}
	// TODO(svayp11): INSERT CLIENT_ID
	pld, err := h.externalClient.Validate(c, req.IdToken, "CLIENT_ID")
	if err != nil {
		return httperrors.Errorf(http.StatusBadRequest, "invalid google token: %w", err)
	}

	s, tx, err := h.clientFactory.NewStorageTx(c, nil)
	if err != nil {
		return xerrors.Errorf("get storage client: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	createUserReq := storage.CreateUserRequest{
		Username:  pld.Claims["name"].(string),
		AvatarURL: pld.Claims["picture"].(string),
	}
	user, err := s.CreateUser(c, &createUserReq)
	if err != nil {
		if errors.Is(err, storage.ErrExists) {
			user, err = s.GetUser(c, &storage.GetUserRequest{Username: createUserReq.Username})
			if err != nil {
				return xerrors.Errorf("get and insert user: %w", err)
			}
			return h.sendAuthDataAndCommit(c, user, tx)
		}
		return xerrors.Errorf("create user: %w", err)
	}

	if err := h.sendAuthDataAndCommit(c, user, tx); err != nil {
		return xerrors.Errorf("%w", err)
	}

	logging.Info(c, "google registration done",
		zap.String("username", user.Username),
		zap.String("user_id", user.ID),
	)
	return nil
}

func (h *Handler) sendAuthDataAndCommit(c *gin.Context, user *storage.User, tx driver.Tx) error {
	token, err := h.tokenGenerator.CreateToken(user)
	if err != nil {
		return xerrors.Errorf("failed to issue token: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return xerrors.Errorf("failed to commit tx: %w", err)
	}

	// TODO(svayp11): set http-only
	c.SetCookie(jwt.AuthCookieName, token, 60*60, "/", "questspace.app", true, false)
	c.JSON(http.StatusOK, user)
	return nil
}
