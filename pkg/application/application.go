package application

import (
	"context"
	"flag"
	"fmt"
	"questspace/pkg/application/errors"
	"time"

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
		return
	}

	if err := SetEnvMode(args.Environment); err != nil {
		fmt.Printf("Failed to set environment mode to %s: %+v", args.Environment, err)
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
		fmt.Printf("Failed to get config from path %s: %+v", args.ConfigsDir, err)
		return
	}

	if err := initFunc(app); err != nil {
		fmt.Printf("Failed to initialize application: %+v", err)
		return
	}
	if err := app.engine.Run(GetAddrFromEnvironment(args.Environment)); err != nil {
		logger.Error("Server error", zap.Error(err))
	}
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
