package authservice

import (
	"context"
	"errors"
	"net/http"

	"github.com/gofrs/uuid"
	"go.uber.org/zap"
	"golang.org/x/xerrors"

	"questspace/internal/hasher"
	"questspace/internal/pgdb"
	"questspace/internal/questspace/authservice/authtypes"
	"questspace/internal/questspace/userservice/usertypes"
	"questspace/pkg/auth/jwt"
	"questspace/pkg/dbnode"
	"questspace/pkg/httperrors"
	"questspace/pkg/logging"
	"questspace/pkg/storage"
)

const (
	defaultURLTmpl = "https://api.dicebear.com/7.x/thumbs/svg?seed="
)

func defaultURL() string {
	return defaultURLTmpl + uuid.Must(uuid.NewV4()).String()
}

type Impl struct {
	clientFactory  pgdb.QuestspaceClientFactory
	imageValidator authtypes.ImageValidator
	pwHasher       hasher.Hasher
	tokenEncoder   jwt.TokenEncoder
}

func NewImpl(
	clientFactory pgdb.QuestspaceClientFactory,
	imageValidator authtypes.ImageValidator,
	pwHasher hasher.Hasher,
	tokenEncoder jwt.TokenEncoder,
) Impl {
	return Impl{
		clientFactory:  clientFactory,
		imageValidator: imageValidator,
		pwHasher:       pwHasher,
		tokenEncoder:   tokenEncoder,
	}
}

func (i *Impl) SignUp(ctx context.Context, req *authtypes.BasicSignUpRequest) (authtypes.Response, error) {
	resp := authtypes.Response{}
	if err := req.Validate(ctx, i.imageValidator); err != nil {
		return resp, err
	}

	createUserReq := storage.CreateUserRequest{
		Username:  req.Username,
		AvatarURL: req.AvatarURL,
	}
	if len(createUserReq.AvatarURL) == 0 {
		createUserReq.AvatarURL = defaultURL()
	}
	var err error
	createUserReq.Password, err = i.pwHasher.HashString(req.Password)
	if err != nil {
		return resp, xerrors.Errorf("hash password: %w", err)
	}

	s, tx, err := i.clientFactory.NewStorageTx(ctx, nil)
	if err != nil {
		return resp, xerrors.Errorf("create tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	err = i.doSignUp(ctx, s, &createUserReq, &resp)
	if err != nil {
		return resp, err
	}

	if err = tx.Commit(); err != nil {
		return resp, xerrors.Errorf("commit tx: %w", err)
	}

	logging.Info(ctx, "basic registration done",
		zap.String("username", resp.User.Username),
		zap.Stringer("user_id", resp.User.ID),
	)
	return resp, nil
}

func (i *Impl) doSignUp(
	ctx context.Context,
	s storage.QuestSpaceStorage,
	req *storage.CreateUserRequest,
	resp *authtypes.Response,
) error {
	user, err := s.CreateUser(ctx, req)
	if err != nil {
		if errors.Is(err, storage.ErrExists) {
			return httperrors.Errorf(http.StatusBadRequest, "user %q already exits", req.Username)
		}
		return xerrors.Errorf("create user: %w", err)
	}

	resp.AccessToken, err = i.tokenEncoder.CreateToken(user)
	if err != nil {
		return xerrors.Errorf("issue new token: %w", err)
	}
	resp.User = usertypes.User{
		ID:        user.ID,
		Username:  user.Username,
		AvatarURL: user.AvatarURL,
	}

	return nil
}

func (i *Impl) SignIn(ctx context.Context, req *authtypes.BasicSignInRequest) (authtypes.Response, error) {
	resp := authtypes.Response{}

	s, err := i.clientFactory.NewStorage(ctx, dbnode.Alive)
	if err != nil {
		return resp, xerrors.Errorf("create storage: %w", err)
	}

	err = i.doSignIn(ctx, s, req, &resp)
	if err != nil {
		return resp, err
	}

	return resp, nil
}

func (i *Impl) doSignIn(
	ctx context.Context,
	s storage.QuestSpaceStorage,
	req *authtypes.BasicSignInRequest,
	resp *authtypes.Response,
) error {
	pwHash, err := s.GetUserPasswordHash(ctx, &storage.GetUserRequest{Username: req.Username})
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return httperrors.Errorf(http.StatusNotFound, "user %q does not exist", req.Username)
		}
		return xerrors.Errorf("lookup user password: %w", err)
	}
	if !i.pwHasher.HasSameHash(req.Password, pwHash) {
		return httperrors.New(http.StatusForbidden, "invalid password")
	}

	user, err := s.GetUser(ctx, &storage.GetUserRequest{Username: req.Username})
	if err != nil {
		return xerrors.Errorf("get user: %w", err)
	}

	resp.AccessToken, err = i.tokenEncoder.CreateToken(user)
	if err != nil {
		return xerrors.Errorf("issue new token: %w", err)
	}
	resp.User = usertypes.User{
		ID:        user.ID,
		Username:  user.Username,
		AvatarURL: user.AvatarURL,
	}

	return nil
}
