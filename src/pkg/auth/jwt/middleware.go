package jwt

import (
	"context"
	"net/http"
	"strings"

	"questspace/pkg/transport"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"questspace/pkg/httperrors"
	"questspace/pkg/logging"
	"questspace/pkg/storage"
)

const AuthCookieName = "qs_user_acc"

type jwtKey struct{}

func middleware(parser Parser, strict bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := getTokenFromRequest(c.Request)
		if token == "" && strict {
			httperrors.WriteErrorResponse(c, httperrors.New(http.StatusUnauthorized, "no credentials found"))
			return
		} else if token == "" {
			c.Next()
			return
		}

		user, err := parser.ParseToken(token)
		if err != nil {
			httperrors.WriteErrorResponse(c, httperrors.WrapWithCode(http.StatusUnauthorized, err))
			return
		}

		logging.AddFieldsToContextLogger(c, zap.Dict("user",
			zap.String("id", user.ID),
			zap.String("username", user.Username),
		))

		userCtx := context.WithValue(c.Request.Context(), jwtKey{}, user)
		c.Request = c.Request.WithContext(userCtx)
		c.Next()
	}
}

func stdMiddleware(parser Parser, strict bool) transport.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := getTokenFromRequest(r)
			if token == "" && strict {
				transport.ServeErrorResponse(r.Context(), w, httperrors.New(http.StatusUnauthorized, "no credentials found"))
				return
			} else if token == "" {
				next.ServeHTTP(w, r)
				return
			}

			user, err := parser.ParseToken(token)
			if err != nil {
				transport.ServeErrorResponse(r.Context(), w, httperrors.WrapWithCode(http.StatusUnauthorized, err))
				return
			}

			logCtx := logging.AddFieldsToContextLogger(r.Context(), zap.Dict("user",
				zap.String("id", user.ID),
				zap.String("username", user.Username),
			))

			userCtx := context.WithValue(logCtx, jwtKey{}, user)
			*r = *r.WithContext(userCtx)
			next.ServeHTTP(w, r)
		})
	}
}

func StdMiddlewareStrict(parser Parser) transport.Middleware {
	return stdMiddleware(parser, true)
}

func StdMiddleware(parser Parser) transport.Middleware {
	return stdMiddleware(parser, false)
}

func AuthMiddlewareStrict(parser Parser) gin.HandlerFunc {
	return middleware(parser, true)
}

func AuthMiddleware(parser Parser) gin.HandlerFunc {
	return middleware(parser, false)
}

func getTokenFromRequest(req *http.Request) string {
	if htk := getFromHeader(req); htk != "" {
		return htk
	}
	if ctk := getFromCookies(req); ctk != "" {
		return ctk
	}
	return ""
}

func getFromHeader(req *http.Request) string {
	jwtHeader := req.Header.Get("Authorization")
	if jwtHeader == "" {
		return ""
	}
	if !strings.HasPrefix(jwtHeader, "Bearer ") && !strings.HasPrefix(jwtHeader, "bearer ") {
		return ""
	}
	tokenStrParts := strings.Split(jwtHeader, " ")
	if len(tokenStrParts) != 2 {
		return ""
	}
	return tokenStrParts[1]
}

func getFromCookies(req *http.Request) string {
	cookie, err := req.Cookie(AuthCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func GetUserFromContext(ctx context.Context) (*storage.User, error) {
	user := ctx.Value(jwtKey{})
	if user == nil {
		return nil, httperrors.New(http.StatusUnauthorized, "no credentials found")
	}
	return user.(*storage.User), nil
}
