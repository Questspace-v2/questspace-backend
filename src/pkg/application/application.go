package application

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/gofor-little/env"
	"go.uber.org/zap"

	"questspace/pkg/application/httperrors"
	"questspace/pkg/application/logging"
)

type applicationArgs struct {
	ConfigsDir  string
	Environment Environment
}

type App struct {
	context context.Context
	engine  *gin.Engine
	logger  *zap.Logger
}

func (a App) Router() *gin.RouterGroup {
	return &a.engine.RouterGroup
}

func (a App) Logger() *zap.Logger {
	return a.logger
}

func Run(initFunc func(app App) error, configHolder interface{}) {
	// TODO(svayp11): configure settings for gin engine
	args, err := getCLIArgs()
	if err != nil {
		fmt.Printf("failed to read environment: %+v", err)
		os.Exit(1)
	}
	logger, err := GetLoggerFromEnvironment(args.Environment)
	if err != nil {
		fmt.Printf("Failed to get logger from environment: %+v", err)
		os.Exit(1)
	}
	defer func() { _ = logger.Sync() }()
	if err := SetEnvMode(args.Environment); err != nil {
		logger.Error("Failed to set environment mode", zap.Stringer("target_mode", &args.Environment), zap.Error(err))
	}

	app := App{context: context.Background(), engine: gin.New()}
	app.engine.ContextWithFallback = true

	_, err = os.Stat(".env")
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		logger.Error("Failed to check .env file")
		os.Exit(1)
	}
	if err == nil {
		if err := env.Load(".env"); err != nil {
			logger.Error("Failed to load environment form .env file", zap.Error(err))
			os.Exit(1)
		}
	} else {
		logger.Warn("Not found .env file")
	}

	app.logger = logger
	app.engine.Use(logging.Middleware(logger), gin.CustomRecovery(logging.RecoveryMiddleware))

	// liveness check
	app.engine.GET("/ping", Ping)

	path := GetConfigFromEnvironment(args.ConfigsDir, args.Environment)
	if err := UnmarshallConfigFromFile(path, configHolder); err != nil {
		logger.Error("Failed to get config from path", zap.String("config_path", args.ConfigsDir), zap.Error(err))
		os.Exit(1)
	}

	if err := initFunc(app); err != nil {
		logger.Error("Failed to initialize application", zap.Error(err))
		os.Exit(1)
	}
	if err := app.engine.Run(GetAddrFromEnvironment(args.Environment)); err != nil {
		logger.Error("Server error", zap.Error(err))
		os.Exit(1)
	}
}

type AppHandlerFunc func(c *gin.Context) error

type AppHandler interface {
	Handle(c *gin.Context) error
}

func AsGinHandler(handler func(c *gin.Context) error) gin.HandlerFunc {
	return func(c *gin.Context) {
		err := handler(c)
		if err != nil {
			httperrors.WriteErrorResponse(c, err)
		}
		logging.Info(c, "new request handled")
	}
}

func getCLIArgs() (applicationArgs, error) {
	args := applicationArgs{}
	flag.StringVar(&args.ConfigsDir, "config", "", "Path to .yaml file with application config")
	flag.Var(&args.Environment, "environment", "Application environment")
	flag.Parse()

	if args.Environment != "" {
		return args, nil
	}
	environ, err := GetEnvironmentFromSystem()
	if err != nil {
		return applicationArgs{}, err
	}
	args.Environment = environ
	return args, nil
}
