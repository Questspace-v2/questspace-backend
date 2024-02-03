package jwt

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"questspace/pkg/application"
	aerrors "questspace/pkg/application/errors"
	"questspace/pkg/storage"
)

func WithJWTMiddleware(parser Parser, handler application.AppHandlerFunc) application.AppHandlerFunc {
	return func(c *gin.Context) error {
		jwtHeader := c.GetHeader("Authorization")
		if jwtHeader == "" {
			return aerrors.NewHttpError(http.StatusForbidden, "no token was provided")
		}
		if !strings.HasPrefix(jwtHeader, "Bearer ") && !strings.HasPrefix(jwtHeader, "bearer ") {
			return aerrors.NewHttpError(http.StatusForbidden, "invalid auth")
		}
		tokenStrParts := strings.Split(jwtHeader, " ")
		if len(tokenStrParts) != 2 {
			return aerrors.NewHttpError(http.StatusForbidden, "invalid header format")
		}
		tokenStr := tokenStrParts[1]
		user, err := parser.ParseToken(tokenStr)
		if err != nil {
			return aerrors.WrapHTTP(http.StatusForbidden, err)
		}

		c.Set("user-creds", user)
		return handler(c)
	}
}

func GetUserFromContext(c *gin.Context) (*storage.User, error) {
	userVal := c.Value("user-creds")
	if userVal == nil {
		return nil, aerrors.NewHttpError(http.StatusForbidden, "no credentials found")
	}

	if user, ok := userVal.(*storage.User); ok {
		return user, nil
	}
	return nil, aerrors.NewHttpError(http.StatusForbidden, "invalid credentials")
}
