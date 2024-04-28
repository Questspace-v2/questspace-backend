package environment

import (
	"os"

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
	default:
		return ":80"
	}
}

const AppEnvironmentEnvKey = "ENVIRONMENT"

func GetEnvironmentFromSystem() (Environment, error) {
	env, ok := os.LookupEnv(AppEnvironmentEnvKey)
	if !ok {
		return Development, nil
	}
	appEnv := new(Environment)
	if err := appEnv.Set(env); err != nil {
		return "", xerrors.Errorf("read app env from environment: %w", err)
	}
	return *appEnv, nil
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
