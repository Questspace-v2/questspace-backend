package auth

import (
	"context"
	"net/http"

	"questspace/internal/questspace/authservice/authtypes"
	"questspace/pkg/transport"
)

//go:generate mockgen -source=auth.go -destination authmock/service.go -package authmock
type AuthService interface {
	SignUp(context.Context, *authtypes.BasicSignUpRequest) (authtypes.Response, error)
	SignIn(context.Context, *authtypes.BasicSignInRequest) (authtypes.Response, error)
}

type RefactoredHandler struct {
	authService AuthService
}

func NewRefactoredHandler(authService AuthService) RefactoredHandler {
	return RefactoredHandler{
		authService: authService,
	}
}

// HandleBasicSignUp handles POST /auth/register request
//
// @Summary	Register new user and return auth data
// @Tags	Auth
// @Param	request	body		authtypes.BasicSignUpRequest	true	"User data to use for sign-up"
// @Success	200		{object}	authtypes.Response
// @Failure	400
// @Failure	415
// @Router	/auth/register [post]
func (h *RefactoredHandler) HandleBasicSignUp(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	req, err := transport.UnmarshalRequestData[authtypes.BasicSignUpRequest](r)
	if err != nil {
		return err
	}
	resp, err := h.authService.SignUp(ctx, &req)
	if err != nil {
		return err
	}
	if err = transport.ServeJSONResponse(w, http.StatusOK, &resp); err != nil {
		return err
	}
	return nil
}

// HandleBasicSignIn handles POST /auth/sign-in request
//
// @Summary	Sign in to user account and return auth data
// @Tags	Auth
// @Param	request	body		authtypes.BasicSignInRequest	true	"Username with password"
// @Success	200		{object}	authtypes.Response
// @Failure	400
// @Failure	403
// @Failure	404
// @Router	/auth/sign-in [post]
func (h *RefactoredHandler) HandleBasicSignIn(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	req, err := transport.UnmarshalRequestData[authtypes.BasicSignInRequest](r)
	if err != nil {
		return err
	}
	resp, err := h.authService.SignIn(ctx, &req)
	if err != nil {
		return err
	}
	if err = transport.ServeJSONResponse(w, http.StatusOK, &resp); err != nil {
		return err
	}
	return nil
}
