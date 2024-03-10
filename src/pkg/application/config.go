package application

import (
	"os"
	"strings"

	"golang.org/x/xerrors"
	"gopkg.in/yaml.v3"
)

func UnmarshallConfigFromFile(path string, config interface{}) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return xerrors.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(content, config); err != nil {
		return xerrors.Errorf("unmarshall config: %w", err)
	}
	return nil
}

func ReadSecret(value string) (string, error) {
	if strings.HasPrefix(value, "env:") {
		val, ok := os.LookupEnv(strings.SplitN(value, ":", 2)[1])
		if ok {
			return val, nil
		}
	}
	secret, err := os.ReadFile(value)
	if err != nil {
		return "", xerrors.Errorf("read secret file: %w", err)
	}
	return string(secret), nil
}
