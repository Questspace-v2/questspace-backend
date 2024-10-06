package google

import (
	"context"
	"net/http"

	"questspace/internal/questspace/authservice/authtypes"
	"questspace/pkg/transport"
)

//go:generate mockgen -source=google.go -destination googlemock/service.go -package googlemock
type GoogleService interface {
	GoogleOAuth(context.Context, *authtypes.GoogleOAuthRequest) (authtypes.Response, error)
}

type RefactoredHandler struct {
	googleService GoogleService
}

func NewRefactoredHandler(googleService GoogleService) RefactoredHandler {
	return RefactoredHandler{
		googleService: googleService,
	}
}

// Handle handles POST /auth/google request
//
// @Summary	Register new or sign in old user using Google OAuth2.0
// @Tags	Auth
// @Param	request	body		authtypes.GoogleOAuthRequest	true	"Google OAuth request"
// @Success	200		{object}	authtypes.Response
// @Failure	400
// @Router	/auth/google [post]
func (h *RefactoredHandler) Handle(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	req, err := transport.UnmarshalRequestData[authtypes.GoogleOAuthRequest](r)
	if err != nil {
		return err
	}

	resp, err := h.googleService.GoogleOAuth(ctx, &req)
	if err != nil {
		return err
	}

	if err = transport.ServeJSONResponse(w, http.StatusOK, &resp); err != nil {
		return err
	}
	return nil
}
