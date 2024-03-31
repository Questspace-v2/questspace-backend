package google

import (
	"context"
	"net/http"

	"go.uber.org/zap"

	"questspace/pkg/application/logging"

	"github.com/gin-gonic/gin"
	"golang.org/x/xerrors"
	"google.golang.org/api/idtoken"

	"questspace/internal/handlers/auth"
	"questspace/internal/handlers/transport"
	pgdb "questspace/internal/pgdb/client"
	"questspace/pkg/application/httperrors"
	"questspace/pkg/auth/jwt"
	"questspace/pkg/storage"
)

type OAuthHandler struct {
	factory        pgdb.QuestspaceClientFactory
	tokenValidator *idtoken.Validator
	jwtParser      jwt.Parser
	config         Config
}

func NewOAuthHandler(f pgdb.QuestspaceClientFactory, tv *idtoken.Validator, p jwt.Parser, cfg Config) *OAuthHandler {
	return &OAuthHandler{
		factory:        f,
		tokenValidator: tv,
		jwtParser:      p,
		config:         cfg,
	}
}

type OAuthRequest struct {
	IDToken string `json:"id_token"`
}

// Handle handles POST /auth/google request
//
// @Summary	Register new or sign in old user using Google OAuth2.0
// @Tags	Auth
// @Param	request	body		google.OAuthRequest	true	"Google OAuth request"
// @Success	200		{object}	auth.Response
// @Failure	400
// @Router	/auth/google [post]
func (o *OAuthHandler) Handle(c *gin.Context) error {
	req, err := transport.UnmarshalRequestData[OAuthRequest](c.Request)
	if err != nil {
		return xerrors.Errorf("%w", err)
	}
	storageReq, err := o.parseToken(c, req.IDToken)
	if err != nil {
		return xerrors.Errorf("%w", err)
	}

	s, tx, err := o.factory.NewStorageTx(c, nil)
	if err != nil {
		return xerrors.Errorf("start tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	user, err := s.CreateOrUpdateByExternalID(c, storageReq)
	if err != nil {
		return xerrors.Errorf("create or update google user: %w", err)
	}
	token, err := o.jwtParser.CreateToken(user)
	if err != nil {
		return xerrors.Errorf("create token: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return xerrors.Errorf("commit tx: %w", err)
	}

	resp := auth.Response{
		User:        user,
		AccessToken: token,
	}
	c.JSON(http.StatusOK, resp)

	logging.Info(c, "google sign-in done",
		zap.String("username", user.Username),
		zap.String("user_id", user.ID),
	)

	return nil
}

func (o *OAuthHandler) parseToken(ctx context.Context, accessToken string) (*storage.CreateOrUpdateRequest, error) {
	payload, err := o.tokenValidator.Validate(ctx, accessToken, o.config.ClientID)
	if err != nil {
		return nil, httperrors.Errorf(http.StatusBadRequest, "bad token: %w", err)
	}

	return &storage.CreateOrUpdateRequest{
		ExternalID: payload.Claims["sub"].(string),
		CreateUserRequest: storage.CreateUserRequest{
			Username:  payload.Claims["email"].(string),
			AvatarURL: payload.Claims["picture"].(string),
		},
	}, nil
}
