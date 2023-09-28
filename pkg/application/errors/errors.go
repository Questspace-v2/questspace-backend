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
		logger.Error("error during handling response", zap.String("errors", c.Errors.String()))
	}
}

var (
	ErrNotFound = xerrors.New("not found")
	ErrInternal = xerrors.New("internal server error")
)

func WriteErrorResponse(c *gin.Context, err error) {
	// TODO(svayp11): More errs to write
	if errors.Is(err, ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": ErrNotFound.Error()})
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"error": ErrInternal.Error()})
	}
}
