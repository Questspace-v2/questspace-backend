package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/zap"
	"golang.org/x/xerrors"

	"questspace/internal/handlers/transport"
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
}

func NewHandler(cf pgdb.QuestspaceClientFactory, f http.Client, h hasher.Hasher, tg jwt.Parser) *Handler {
	return &Handler{
		clientFactory:  cf,
		fetcher:        f,
		pwHasher:       h,
		tokenGenerator: tg,
	}
}

type Response struct {
	User        *storage.User `json:"user"`
	AccessToken string        `json:"access_token"`
}

// HandleBasicSignUp handles POST /auth/register request
//
// @Summary	Register new user and return auth data
// @Tags	Auth
// @Param	request	body		storage.CreateUserRequest	true	"Create user request"
// @Success	200		{object}	auth.Response
// @Failure	400
// @Failure	415
// @Router	/auth/register [post]
func (h *Handler) HandleBasicSignUp(c *gin.Context) error {
	req, err := transport.UnmarshalRequestData[storage.CreateUserRequest](c.Request)
	if err != nil {
		return xerrors.Errorf("%w", err)
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
	user, err := s.CreateUser(c, req)
	if err != nil {
		if errors.Is(err, storage.ErrExists) {
			return httperrors.Errorf(http.StatusBadRequest, "user %q already exits", req.Username)
		}
		return xerrors.Errorf("create user: %w", err)
	}
	resp := Response{
		User: user,
	}
	resp.AccessToken, err = h.tokenGenerator.CreateToken(user)
	if err != nil {
		return xerrors.Errorf("issue token: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return xerrors.Errorf("commit tx: %w", err)
	}
	c.JSON(http.StatusOK, resp)

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
// @Summary	Sign in to user account and return auth data
// @Tags	Auth
// @Param	request	body		auth.SignInRequest	true	"Sign in request"
// @Success	200		{object}	auth.Response
// @Failure	400
// @Failure	403
// @Failure	404
// @Router	/auth/sign-in [post]
func (h *Handler) HandleBasicSignIn(c *gin.Context) error {
	req, err := transport.UnmarshalRequestData[SignInRequest](c.Request)
	if err != nil {
		return xerrors.Errorf("%w", err)
	}

	s, err := h.clientFactory.NewStorage(c, dbnode.Alive)
	if err != nil {
		return xerrors.Errorf("get storage client: %w", err)
	}
	pwHash, err := s.GetUserPasswordHash(c, &storage.GetUserRequest{Username: req.Username})
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return httperrors.Errorf(http.StatusNotFound, "user %q does not exist", req.Username)
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
	resp := Response{
		User:        user,
		AccessToken: token,
	}
	c.JSON(http.StatusOK, resp)
	return nil
}
