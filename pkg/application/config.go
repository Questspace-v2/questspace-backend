package application

import (
	"os"

	"golang.org/x/xerrors"
	"gopkg.in/yaml.v3"
)

func UnmarshallConfigFromFile(path string, config interface{}) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return xerrors.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(content, config); err != nil {
		return xerrors.Errorf("failed to unmarshall config: %w", err)
	}
	return nil
}
