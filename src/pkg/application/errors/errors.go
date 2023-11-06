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

var (
	ErrBadRequest = xerrors.New("bad request")
	ErrNotFound   = xerrors.New("not found")
	ErrValidation = xerrors.New("validation error")
	ErrInternal   = xerrors.New("internal server error")
)

func WriteErrorResponse(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrBadRequest):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		break
	case errors.Is(err, ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		break
	case errors.Is(err, ErrValidation):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		break
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": ErrInternal.Error()})
		_ = c.Error(err)
		break
	}
}
