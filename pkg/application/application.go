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
	ConfigPath  string
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
	app.logger = logger
	app.engine.Use(ginzap.Ginzap(app.logger, time.RFC3339, false))
	app.engine.Use(func(c *gin.Context) {
		errors.ErrorHandler(logger)(c)
	})

	// liveness check
	app.engine.GET("/ping", Ping)

	if err := UnmarshallConfigFromFile(args.ConfigPath, configHolder); err != nil {
		fmt.Printf("Failed to get config from path %s: %+v", args.ConfigPath, err)
		return
	}

	if err := initFunc(app); err != nil {
		fmt.Printf("Failed to initialize application: %+v", err)
		return
	}
	app.engine.Run(GetAddrFromEnvironment(args.Environment))
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
	flag.StringVar(&args.ConfigPath, "config", "", "Path to .yaml file with application config")
	flag.Var(&args.Environment, "environment", "Application environment")
	flag.Parse()

	// TODO(svayp11): check env variables
	if args.Environment == "" {
		args.Environment = Development
	}
	return args
}
