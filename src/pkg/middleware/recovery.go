package middleware

import (
	"net/http"

	"go.uber.org/zap"

	"questspace/pkg/logging"
	"questspace/pkg/transport"
)

func Recovery() transport.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if cause := recover(); cause != nil {
					logging.Error(r.Context(), "panic during handling request", zap.Any("cause", cause))
					transport.ServeText(w, http.StatusInternalServerError, "internal server error")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
