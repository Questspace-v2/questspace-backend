package jwt

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"questspace/pkg/application"
	"questspace/pkg/application/httperrors"
	"questspace/pkg/application/logging"
	"questspace/pkg/storage"
)

const AuthCookieName = "qs_user_acc"

type jwtKey struct{}

func middleware(parser Parser, strict bool) gin.HandlerFunc {
	return application.AsGinHandler(func(c *gin.Context) error {
		token := getTokenFromRequest(c.Request)
		if token == "" && strict {
			return httperrors.New(http.StatusUnauthorized, "no credentials found")
		} else if token == "" {
			c.Next()
			return nil
		}

		user, err := parser.ParseToken(token)
		if err != nil {
			return httperrors.WrapWithCode(http.StatusUnauthorized, err)
		}

		logging.AddFieldsToContextLogger(c, zap.Dict("user",
			zap.String("id", user.ID),
			zap.String("username", user.Username),
		))

		userCtx := context.WithValue(c.Request.Context(), jwtKey{}, user)
		c.Request = c.Request.WithContext(userCtx)
		c.Next()
		return nil
	})
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
