package application

import (
	"context"
	"flag"
	"fmt"
	"os"
	"questspace/pkg/application/errors"
	"time"

	"github.com/gofor-little/env"

	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
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
	app := App{context: context.Background(), engine: gin.New()}
	args := getCLIArgs()

	logger, err := GetLoggerFromEnvironment(args.Environment)
	if err != nil {
		fmt.Printf("Failed to get logger from environment: %+v", err)
		os.Exit(1)
	}

	if err := env.Load(".env"); err != nil {
		logger.Error("Failed to load environment form .env file", zap.Error(err))
		os.Exit(1)
	}

	if err := SetEnvMode(args.Environment); err != nil {
		logger.Error("Failed to set environment mode", zap.Stringer("target_mode", &args.Environment), zap.Error(err))
	}

	app.logger = logger
	app.engine.Use(ginzap.Ginzap(app.logger, time.RFC3339, false))
	app.engine.Use(func(c *gin.Context) {
		errors.ErrorHandler(logger)(c)
	})
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

type AppHandler interface {
	Handle(c *gin.Context) error
}

func AsGinHandler(handler func(c *gin.Context) error) gin.HandlerFunc {
	return func(c *gin.Context) {
		err := handler(c)
		if err != nil {
			_ = c.Error(err)
			errors.WriteErrorResponse(c, err)
		}
	}
}

func getCLIArgs() applicationArgs {
	args := applicationArgs{}
	flag.StringVar(&args.ConfigsDir, "config", "", "Path to .yaml file with application config")
	flag.Var(&args.Environment, "environment", "Application environment")
	flag.Parse()

	// TODO(svayp11): check env variables
	if args.Environment == "" {
		args.Environment = Development
	}
	return args
}
