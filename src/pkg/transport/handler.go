package transport

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"go.uber.org/zap"

	"questspace/pkg/httperrors"
	"questspace/pkg/logging"
)

type AppHandlerFunc func(ctx context.Context, w http.ResponseWriter, r *http.Request) error

func ServeErrorResponse(ctx context.Context, w http.ResponseWriter, err error) {
	if httpErr := new(httperrors.HTTPError); errors.As(err, &httpErr) {
		ServeErr(w, httpErr.Code, httpErr)
		logging.Warn(ctx, "user error",
			zap.String("status_str", http.StatusText(httpErr.Code)),
			zap.Int("status", httpErr.Code),
			zap.String("error_trace", fmt.Sprintf("%+v", httpErr.Unwrap())),
		)
		return
	}
	logging.Error(ctx, "error handling request", zap.String("error_trace", fmt.Sprintf("%+v", err)))
	ServeText(w, http.StatusInternalServerError, "internal server error")
}

func WrapCtxErr(h AppHandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		err := h(ctx, w, r)
		if err != nil {
			ServeErrorResponse(ctx, w, err)
			return
		}
		logging.Info(ctx, "new request handled")
	})
}
