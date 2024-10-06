package authtypes

import (
	"context"
	"net/http"

	"questspace/internal/questspace/userservice/usertypes"
	"questspace/pkg/httperrors"
)

type ImageValidator interface {
	ValidateImageURL(ctx context.Context, imageURL string) error
}

type BasicSignUpRequest struct {
	Username  string `json:"username" example:"svayp11"`
	Password  string `json:"password" example:"12345"`
	AvatarURL string `json:"avatar_url" example:"https://api.dicebear.com/7.x/thumbs/svg"`
}

func (req *BasicSignUpRequest) Validate(ctx context.Context, imageValidator ImageValidator) error {
	if err := imageValidator.ValidateImageURL(ctx, req.AvatarURL); err != nil {
		return err
	}
	if len(req.Password) == 0 {
		return httperrors.New(http.StatusBadRequest, "password cannot be empty")
	}
	return nil
}

type BasicSignInRequest struct {
	Username string `json:"username" example:"svayp11"`
	Password string `json:"password" example:"12345"`
}

type GoogleOAuthRequest struct {
	IDToken string `json:"id_token"`
}

type Response struct {
	User        usertypes.User `json:"user"`
	AccessToken string         `json:"access_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"`
}
