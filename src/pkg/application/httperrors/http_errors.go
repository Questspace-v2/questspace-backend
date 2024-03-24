package httperrors

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/xerrors"

	"questspace/pkg/application/logging"
)

var _ error = &HTTPError{}

type HTTPError struct {
	Code int
	err  error
}

func New(httpCode int, msg string) *HTTPError {
	return &HTTPError{Code: httpCode, err: xerrors.New(msg)}
}

func Errorf(httpCode int, tmpl string, args ...interface{}) *HTTPError {
	return &HTTPError{Code: httpCode, err: xerrors.Errorf(tmpl, args...)}
}

func WrapWithCode(httpCode int, err error) error {
	return &HTTPError{Code: httpCode, err: xerrors.Errorf("%w", err)}
}

func (e *HTTPError) Error() string {
	return e.err.Error()
}

func (e *HTTPError) Unwrap() error {
	return e.err
}

func WriteErrorResponse(c *gin.Context, err error) {
	httpErr := &HTTPError{}
	if errors.As(err, &httpErr) {
		c.JSON(httpErr.Code, gin.H{"error": httpErr.Error()})
		return
	}
	logging.Error(c, "error handling request", zap.Error(err))
	c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
}
