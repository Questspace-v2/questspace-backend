package httperrors

import (
	"errors"
	"fmt"
	"net/http"

	"questspace/pkg/logging"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/xerrors"
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
		logging.Warn(c, "user error",
			zap.String("status_str", http.StatusText(httpErr.Code)),
			zap.Int("status", httpErr.Code),
			zap.String("error_trace", fmt.Sprintf("%+v", httpErr.err)),
		)
		return
	}
	logging.Error(c, "error handling request", zap.String("error_trace", fmt.Sprintf("%+v", err)))
	c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
}
