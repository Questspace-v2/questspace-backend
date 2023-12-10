package errors

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/xerrors"
)

func ErrorHandler(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 {
			return
		}
		logger.Error("error during handling request", zap.String("errors", c.Errors.String()))
	}
}

var _ error = &httpError{}

type httpError struct {
	httpCode int
	error
}

func NewHttpError(httpCode int, tmpl string, args ...interface{}) error {
	return &httpError{httpCode: httpCode, error: xerrors.Errorf(tmpl, args...)}
}

func WrapHTTP(httpCode int, err error) error {
	return &httpError{httpCode: httpCode, error: err}
}

func (e *httpError) Error() string {
	return e.error.Error()
}

func WriteErrorResponse(c *gin.Context, err error) {
	httpErr := &httpError{}
	if errors.As(err, &httpErr) {
		c.JSON(httpErr.httpCode, gin.H{"error": httpErr.Error()})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
}
