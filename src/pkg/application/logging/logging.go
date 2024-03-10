package logging

import (
	"fmt"

	"github.com/gofrs/uuid"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const loggerKey = "app-logger"

func Middleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger = logger.With(zap.String("response_id", uuid.Must(uuid.NewV4()).String()))
		c.Set(loggerKey, logger)
		c.Next()
	}
}

func AddFieldsToContextLogger(c *gin.Context, fields ...zap.Field) {
	logger := c.MustGet(loggerKey).(*zap.Logger)
	logger = logger.With(fields...)
	c.Set(loggerKey, logger)
}

func GetLogger(c *gin.Context) *zap.Logger {
	return c.MustGet(loggerKey).(*zap.Logger)
}

func log(c *gin.Context, lvl zapcore.Level, msg string, fields ...zap.Field) {
	logger := c.MustGet(loggerKey).(*zap.Logger)
	logger.Log(lvl, msg, fields...)
}

func Debug(c *gin.Context, msg string, fields ...zap.Field) {
	log(c, zap.DebugLevel, msg, fields...)
}

func Info(c *gin.Context, msg string, fields ...zap.Field) {
	log(c, zap.InfoLevel, msg, fields...)
}

func Warn(c *gin.Context, msg string, fields ...zap.Field) {
	log(c, zap.WarnLevel, msg, fields...)
}

func Error(c *gin.Context, msg string, fields ...zap.Field) {
	log(c, zap.ErrorLevel, msg, fields...)
}

func Panic(c *gin.Context, msg string, fields ...zap.Field) {
	log(c, zap.PanicLevel, msg, fields...)
}

func Debugf(c *gin.Context, msg string, params ...interface{}) {
	log(c, zap.DebugLevel, fmt.Sprintf(msg, params...))
}

func Infof(c *gin.Context, msg string, params ...interface{}) {
	log(c, zap.InfoLevel, fmt.Sprintf(msg, params...))
}

func Warnf(c *gin.Context, msg string, params ...interface{}) {
	log(c, zap.WarnLevel, fmt.Sprintf(msg, params...))
}

func Errorf(c *gin.Context, msg string, params ...interface{}) {
	log(c, zap.ErrorLevel, fmt.Sprintf(msg, params...))
}

func Panicf(c *gin.Context, msg string, params ...interface{}) {
	log(c, zap.PanicLevel, fmt.Sprintf(msg, params...))
}
