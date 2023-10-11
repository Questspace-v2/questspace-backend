package application

import (
	"path"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/xerrors"
)

type Environment string

func (e *Environment) String() string {
	return string(*e)
}

func (e *Environment) Set(s string) error {
	switch env := Environment(s); env {
	case Development, Production, DockerDevelopment:
		*e = env
		return nil
	default:
		return xerrors.Errorf("unexpected environment name: %s", env)
	}
}

const (
	Development       Environment = "dev"
	Production        Environment = "prod"
	DockerDevelopment Environment = "docker-dev"
)

func GetAddrFromEnvironment(env Environment) string {
	switch env {
	case Development:
		return "localhost:8080"
	case DockerDevelopment:
		return ":8080"
	case Production:
		return ":80"
	default:
		return ":80"
	}
}

func GetLoggerFromEnvironment(env Environment) (*zap.Logger, error) {
	switch env {
	case Development, DockerDevelopment:
		return zap.NewDevelopment()
	case Production:
		return zap.NewProduction()
	default:
		return nil, xerrors.Errorf("unexpected environment name: %s", env)
	}
}

func GetConfigFromEnvironment(dir string, env Environment) string {
	return path.Join(dir, string(env)+".yaml")
}

func SetEnvMode(env Environment) error {
	switch env {
	case Development, DockerDevelopment:
		gin.SetMode(gin.DebugMode)
		return nil
	case Production:
		gin.SetMode(gin.ReleaseMode)
		return nil
	default:
		return xerrors.Errorf("unimplemented environment: %s", env)
	}
}
