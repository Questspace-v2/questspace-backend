package app

import (
	"context"
	"errors"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/gofor-little/env"
	"go.uber.org/zap"
	"golang.org/x/xerrors"

	"questspace/pkg/environment"
	"questspace/pkg/middleware"
	"questspace/pkg/transport"
)

var (
	configs string
	environ environment.Environment
)

type App struct {
	router   *transport.Router
	logger   *zap.Logger
	cleanups []func() error
}

func (a *App) Router() *transport.Router {
	return a.router
}

func (a *App) Logger() *zap.Logger {
	return a.logger
}

func (a *App) Environment() environment.Environment {
	return environ
}

func NewApp() *App {
	err := getCLIArgs()
	if err != nil {
		log.Fatalf("Failed to read environment: %v", err)
	}
	logger, err := environment.GetLoggerFromEnvironment(environ)
	if err != nil {
		log.Fatalf("Failed to get logger from environment: %v", err)
	}
	defer func() { _ = logger.Sync() }()

	//app := &App{router: transport.NewRouter()}

	_, err = os.Stat(".env")
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Fatalf("Failed to check .env file: %v", err)
	}
	if err == nil {
		if err := env.Load(".env"); err != nil {
			log.Fatalf("Failed to load environment from .env file: %v", err)
		}
	} else {
		logger.Debug("Not found .env file")
	}

	router := transport.NewRouter()
	router.Use(middleware.CtxLog(logger), middleware.Recovery())

	// liveness check
	router.H().GET("/ping", http.HandlerFunc(Ping))
	return &App{
		router: router,
		logger: logger,
	}
}

func (a *App) GetConfig() (*Config, error) {
	cfgPath := path.Join(configs, environ.String()+".yaml")
	cfg, err := UnmarshallConfigFromFile(cfgPath)
	if err != nil {
		return nil, xerrors.Errorf("unmarshal config: %w", err)
	}
	return cfg, nil
}

func (a *App) Cleanup(c func() error) {
	a.cleanups = append(a.cleanups, c)
}

func (a *App) Close() error {
	errs := make([]error, 0, len(a.cleanups))
	for _, c := range a.cleanups {
		if err := c(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (a *App) Run(ctx context.Context) error {
	addr := environment.GetAddrFromEnvironment(environ)
	srv := http.Server{
		Handler:     a.router,
		Addr:        addr,
		ReadTimeout: time.Second * 10,
		ConnContext: func(srvCtx context.Context, _ net.Conn) context.Context {
			baseCtx := context.WithValue(ctx, http.ServerContextKey, srvCtx.Value(http.ServerContextKey))
			return baseCtx
		},
	}

	shutdown, listen := make(chan error), make(chan error)
	go func() {
		<-ctx.Done()

		timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		shutdownErr := srv.Shutdown(timeoutCtx)
		if shutdownErr != nil {
			shutdown <- shutdownErr
		} else {
			close(shutdown)
		}
	}()
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			listen <- err
		} else {
			close(listen)
		}
	}()
	if err := <-shutdown; err != nil {
		return err
	}
	return nil
}

func getCLIArgs() error {
	flag.StringVar(&configs, "config", "", "Path to .yaml file with application config")
	flag.Var(&environ, "environment", "Application environment")
	flag.Parse()

	if environ != "" {
		return nil
	}
	sysEnv, err := environment.GetEnvironmentFromSystem()
	if err != nil {
		return err
	}
	environ = sysEnv
	return nil
}
