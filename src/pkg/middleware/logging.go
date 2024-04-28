package middleware

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"questspace/pkg/logging"

	"questspace/pkg/transport"

	"github.com/gofrs/uuid"
	"go.uber.org/zap"
)

var restrictedHeaders = map[string]struct{}{
	"authorization": {},
	"cookie":        {},
	"cookie2":       {},
}

func CtxLog(logger *zap.Logger) transport.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqId := uuid.Must(uuid.NewV4()).String() + "-" + strconv.FormatInt(time.Now().UTC().Unix(), 10)
			fields := []zap.Field{
				zap.String("request_id", reqId),
			}

			var headers []zap.Field
			for name, values := range r.Header {
				headerNameLower := strings.ToLower(name)
				if _, ok := restrictedHeaders[headerNameLower]; ok {
					headers = append(headers, zap.String(headerNameLower, "***"))
					continue
				}
				headers = append(headers, zap.String(headerNameLower, strings.Join(values, ", ")))
			}

			fields = append(fields,
				zap.String("uri", r.URL.RequestURI()),
				zap.String("method", r.Method),
				zap.Dict("headers", headers...),
			)

			ctxLogger := logger.With(fields...)
			logCtx := logging.WithLogger(r.Context(), ctxLogger)
			*r = *r.WithContext(logCtx)

			next.ServeHTTP(w, r)
		})
	}
}
