package logging

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type loggerKey struct{}

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
			headers = append(headers, zap.String(headerNameLower, strings.Join(values, ", ")))
		}

		fields = append(fields,
			zap.String("uri", req.URL.RequestURI()),
			zap.Dict("headers", headers...),
		)

		logger = logger.With(fields...)
		WithLogger(c, logger)
		c.Next()
	}
}

func RecoveryMiddleware(c *gin.Context, err any) {
	Error(c, "panic during handling request", zap.Any("cause", err))
	c.Status(http.StatusInternalServerError)
}

func AddFieldsToContextLogger(ctx context.Context, fields ...zap.Field) context.Context {
	logger := GetLogger(ctx)
	if logger == nil {
		_, _ = fmt.Fprintf(os.Stderr, "Logger not found. Skipping fields...\n")
		return ctx
	}
	logger = logger.With(fields...)
	return WithLogger(ctx, logger)
}

func WithLogger(ctx context.Context, logger *zap.Logger) context.Context {
	if c, ok := ctx.(*gin.Context); ok {
		logCtx := setCtxLogger(c.Request.Context(), logger)
		c.Request = c.Request.WithContext(logCtx)
		return c
	}
	return setCtxLogger(ctx, logger)
}

func setCtxLogger(ctx context.Context, logger *zap.Logger) context.Context {
	logCtx := context.WithValue(ctx, loggerKey{}, logger)
	return logCtx
}

func GetLogger(ctx context.Context) *zap.Logger {
	val := ctx.Value(loggerKey{})
	if logger, ok := val.(*zap.Logger); ok {
		return logger
	}
	return nil
}

func logMessage(ctx context.Context, lvl zapcore.Level, msg string, fields ...zap.Field) {
	logger := GetLogger(ctx)
	if logger == nil {
		_, _ = fmt.Fprintf(os.Stderr, "Logger not found. Skipping message...\n")
		return
	}
	logger.Log(lvl, msg, fields...)
}

func Debug(ctx context.Context, msg string, fields ...zap.Field) {
	logMessage(ctx, zap.DebugLevel, msg, fields...)
}

func Info(ctx context.Context, msg string, fields ...zap.Field) {
	logMessage(ctx, zap.InfoLevel, msg, fields...)
}

func Warn(ctx context.Context, msg string, fields ...zap.Field) {
	logMessage(ctx, zap.WarnLevel, msg, fields...)
}

func Error(ctx context.Context, msg string, fields ...zap.Field) {
	logMessage(ctx, zap.ErrorLevel, msg, fields...)
}

func Panic(ctx context.Context, msg string, fields ...zap.Field) {
	logMessage(ctx, zap.PanicLevel, msg, fields...)
}

func Debugf(ctx context.Context, msg string, params ...interface{}) {
	logMessage(ctx, zap.DebugLevel, fmt.Sprintf(msg, params...))
}

func Infof(ctx context.Context, msg string, params ...interface{}) {
	logMessage(ctx, zap.InfoLevel, fmt.Sprintf(msg, params...))
}

func Warnf(ctx context.Context, msg string, params ...interface{}) {
	logMessage(ctx, zap.WarnLevel, fmt.Sprintf(msg, params...))
}

func Errorf(ctx context.Context, msg string, params ...interface{}) {
	logMessage(ctx, zap.ErrorLevel, fmt.Sprintf(msg, params...))
}

func Panicf(ctx context.Context, msg string, params ...interface{}) {
	logMessage(ctx, zap.PanicLevel, fmt.Sprintf(msg, params...))
}
