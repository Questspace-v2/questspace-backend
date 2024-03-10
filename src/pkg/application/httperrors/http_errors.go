package httperrors

import (
	"errors"
	"net/http"

	"questspace/pkg/application/logging"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/xerrors"
)

var _ error = &HTTPError{}

type HTTPError struct {
	httpCode int
	err      error
}

func New(httpCode int, msg string) *HTTPError {
	return &HTTPError{httpCode: httpCode, err: xerrors.New(msg)}
}

func Errorf(httpCode int, tmpl string, args ...interface{}) *HTTPError {
	return &HTTPError{httpCode: httpCode, err: xerrors.Errorf(tmpl, args...)}
}

func WrapWithCode(httpCode int, err error) error {
	return &HTTPError{httpCode: httpCode, err: xerrors.Errorf("%w", err)}
}

func (e *HTTPError) Error() string {
	return e.err.Error()
}

func (e *HTTPError) Unwrap() error {
	return e.err
}

func WriteErrorResponse(c *gin.Context, err error) {
	logging.Error(c, "error handling request", zap.Error(err))
	httpErr := &HTTPError{}
	if errors.As(err, &httpErr) {
		c.JSON(httpErr.httpCode, gin.H{"error": httpErr.Error()})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
}
