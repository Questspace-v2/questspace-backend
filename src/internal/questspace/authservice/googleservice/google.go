package googleservice

import (
	"context"
	"math/rand"
	"net/http"
	"strconv"

	"golang.org/x/xerrors"
	"google.golang.org/api/idtoken"

	"questspace/internal/pgdb"
	"questspace/internal/questspace/authservice/authtypes"
	"questspace/internal/questspace/userservice/usertypes"
	"questspace/pkg/auth/jwt"
	"questspace/pkg/httperrors"
	"questspace/pkg/storage"
)

//go:generate mockgen -source=google.go -destination idtokenmock/validator.go -package idtokenmock
type TokenValidator interface {
	Validate(ctx context.Context, idToken string, audience string) (*idtoken.Payload, error)
}

type Auth struct {
	clientFactory    pgdb.QuestspaceClientFactory
	tokenEncoder     jwt.TokenEncoder
	idTokenValidator TokenValidator
	config           *Config
}

func NewAuth(
	clientFactory pgdb.QuestspaceClientFactory,
	tokenEncoder jwt.TokenEncoder,
	idTokenValidator TokenValidator,
	config *Config,
) Auth {
	return Auth{
		clientFactory:    clientFactory,
		tokenEncoder:     tokenEncoder,
		idTokenValidator: idTokenValidator,
		config:           config,
	}
}

func (a *Auth) GoogleOAuth(ctx context.Context, req *authtypes.GoogleOAuthRequest) (authtypes.Response, error) {
	resp := authtypes.Response{}
	oauthReq, err := a.parseToken(ctx, req.IDToken)
	if err != nil {
		return resp, err
	}

	s, tx, err := a.clientFactory.NewStorageTx(ctx, nil)
	if err != nil {
		return resp, xerrors.Errorf("start tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	err = a.doGoogleOAuth(ctx, s, &oauthReq, &resp)
	if err != nil {
		return resp, err
	}

	if err = tx.Commit(); err != nil {
		return resp, xerrors.Errorf("commit tx: %w", err)
	}
	return resp, nil
}

func (a *Auth) parseToken(ctx context.Context, idToken string) (storage.CreateOrUpdateRequest, error) {
	payload, err := a.idTokenValidator.Validate(ctx, idToken, a.config.ClientID)
	if err != nil {
		return storage.CreateOrUpdateRequest{}, httperrors.Errorf(http.StatusBadRequest, "bad token: %w", err)
	}
	randNum := rand.Int() //nolint:gosec

	return storage.CreateOrUpdateRequest{
		ExternalID: payload.Claims["sub"].(string),
		CreateUserRequest: storage.CreateUserRequest{
			Username:  "user-" + strconv.Itoa(randNum),
			AvatarURL: payload.Claims["picture"].(string),
		},
	}, nil
}

func (a *Auth) doGoogleOAuth(
	ctx context.Context,
	s storage.QuestSpaceStorage,
	req *storage.CreateOrUpdateRequest,
	resp *authtypes.Response,
) error {
	user, err := s.CreateOrUpdateByExternalID(ctx, req)
	if err != nil {
		return xerrors.Errorf("create or update google user: %w", err)
	}
	token, err := a.tokenEncoder.CreateToken(user)
	if err != nil {
		return xerrors.Errorf("create token: %w", err)
	}

	resp.User = usertypes.User{
		ID:        user.ID,
		Username:  user.Username,
		AvatarURL: user.AvatarURL,
	}
	resp.AccessToken = token

	return nil
}
