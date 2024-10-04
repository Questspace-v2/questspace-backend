package app

import (
	"os"

	"golang.org/x/xerrors"
	"gopkg.in/yaml.v3"

	"questspace/internal/handlers/teams"
	"questspace/internal/images"
	"questspace/internal/pgdb/pgconfig"
	"questspace/internal/questspace/authservice/googleservice"
	"questspace/pkg/auth/jwt"
	"questspace/pkg/cors"
)

type Config struct {
	DB        pgconfig.Config      `yaml:"db"`
	HashCost  int                  `yaml:"hash-cost"`
	CORS      cors.Config          `yaml:"cors"`
	JWT       jwt.Config           `yaml:"jwt"`
	Teams     teams.Config         `yaml:"teams"`
	Google    googleservice.Config `yaml:"google-oauth"`
	Validator images.Config        `yaml:"validator"`
}

func UnmarshallConfigFromFile(path string) (*Config, error) {
	config := &Config{}

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, xerrors.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(content, config); err != nil {
		return nil, xerrors.Errorf("unmarshall config: %w", err)
	}
	return config, nil
}
