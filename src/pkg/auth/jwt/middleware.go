package jwt

import (
	"net/http"
	"strings"

	"go.uber.org/zap"

	"questspace/pkg/application/logging"

	"github.com/gofrs/uuid"

	"github.com/gin-gonic/gin"

	"questspace/pkg/application"
	"questspace/pkg/application/httperrors"
	"questspace/pkg/storage"
)

const AuthCookieName = "qs_user_acc"

var userCredsKey = ""

func init() {
	userCredsKey = "user-creds" + uuid.Must(uuid.NewV4()).String()
}

func WithJWTMiddleware(parser Parser, handler application.AppHandlerFunc) application.AppHandlerFunc {
	return func(c *gin.Context) error {
		var token string
		if htk := getFromHeader(c); htk != "" {
			token = htk
		} else if ctk := getFromCookies(c); ctk != "" {
			token = ctk
		} else {
			return httperrors.Errorf(http.StatusForbidden, "no auth token was provided")
		}
		user, err := parser.ParseToken(token)
		if err != nil {
			return httperrors.WrapWithCode(http.StatusForbidden, err)
		}

		logging.AddFieldsToContextLogger(c, zap.Dict("user",
			zap.String("id", user.ID),
			zap.String("username", user.Username),
		))

		c.Set(userCredsKey, user)
		return handler(c)
	}
}

func getFromHeader(c *gin.Context) string {
	jwtHeader := c.GetHeader("Authorization")
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

func getFromCookies(c *gin.Context) string {
	cookie, err := c.Cookie(AuthCookieName)
	if err != nil {
		return ""
	}
	return cookie
}

func GetUserFromContext(c *gin.Context) (*storage.User, error) {
	userVal := c.Value(userCredsKey)
	if userVal == nil {
		return nil, httperrors.Errorf(http.StatusUnauthorized, "no credentials found")
	}

	if user, ok := userVal.(*storage.User); ok {
		return user, nil
	}
	return nil, httperrors.Errorf(http.StatusUnauthorized, "invalid credentials")
}
