package logging

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const loggerKey = "app-logger"

var restrictedHeaders = map[string]struct{}{
	"authorization": {},
	"cookie":        {},
	"cookie2":       {},
}

func Middleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		reqId := uuid.Must(uuid.NewV4()).String() + "-" + strconv.FormatInt(time.Now().UTC().Unix(), 10)
		fields := []zap.Field{
			zap.String("request_id", reqId),
		}

		req := c.Request
		var headers []zap.Field
		for name, values := range req.Header {
			headerNameLower := strings.ToLower(name)
			if _, ok := restrictedHeaders[headerNameLower]; ok {
				headers = append(headers, zap.String(headerNameLower, "***"))
				continue
			}
			for _, val := range values {
				headers = append(headers, zap.String(headerNameLower, val))
			}
		}
		fields = append(fields,
			zap.String("uri", req.RequestURI+"?"+req.URL.RawQuery),
			zap.Dict("headers", headers...),
		)

		logger = logger.With(fields...)
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

func DebugCtx(ctx context.Context, msg string, fields ...zap.Field) {
	c, ok := ctx.(*gin.Context)
	if !ok {
		return
	}
	log(c, zap.DebugLevel, msg, fields...)
}

func InfoCtx(ctx context.Context, msg string, fields ...zap.Field) {
	c, ok := ctx.(*gin.Context)
	if !ok {
		return
	}
	log(c, zap.InfoLevel, msg, fields...)
}

func WarnCtx(ctx context.Context, msg string, fields ...zap.Field) {
	c, ok := ctx.(*gin.Context)
	if !ok {
		return
	}
	log(c, zap.WarnLevel, msg, fields...)
}

func ErrorCtx(ctx context.Context, msg string, fields ...zap.Field) {
	c, ok := ctx.(*gin.Context)
	if !ok {
		return
	}
	log(c, zap.ErrorLevel, msg, fields...)
}

func PanicCtx(ctx context.Context, msg string, fields ...zap.Field) {
	c, ok := ctx.(*gin.Context)
	if !ok {
		return
	}
	log(c, zap.PanicLevel, msg, fields...)
}

func DebugCtxf(ctx context.Context, msg string, params ...interface{}) {
	c, ok := ctx.(*gin.Context)
	if !ok {
		return
	}
	log(c, zap.DebugLevel, fmt.Sprintf(msg, params...))
}

func InfoCtxf(ctx context.Context, msg string, params ...interface{}) {
	c, ok := ctx.(*gin.Context)
	if !ok {
		return
	}
	log(c, zap.InfoLevel, fmt.Sprintf(msg, params...))
}

func WarnCtxf(ctx context.Context, msg string, params ...interface{}) {
	c, ok := ctx.(*gin.Context)
	if !ok {
		return
	}
	log(c, zap.WarnLevel, fmt.Sprintf(msg, params...))
}

func ErrorCtxf(ctx context.Context, msg string, params ...interface{}) {
	c, ok := ctx.(*gin.Context)
	if !ok {
		return
	}
	log(c, zap.ErrorLevel, fmt.Sprintf(msg, params...))
}

func PanicCtxf(ctx context.Context, msg string, params ...interface{}) {
	c, ok := ctx.(*gin.Context)
	if !ok {
		return
	}
	log(c, zap.PanicLevel, fmt.Sprintf(msg, params...))
}
