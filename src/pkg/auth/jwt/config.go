package jwt

import (
	"gopkg.in/yaml.v3"

	"questspace/pkg/application"
)

type Config struct {
	Secret string `yaml:"secret"`
}

func (c *Config) UnmarshalYAML(value *yaml.Node) error {
	type configAlias Config
	temp := &configAlias{}

	if err := value.Decode(temp); err != nil {
		return err
	}

	keySec, err := application.ReadSecret(temp.Secret)
	if err != nil {
		return err
	}
	temp.Secret = keySec

	*c = Config(*temp)
	return nil
}

func (c *Config) GetEncryptionKey() []byte {
	return []byte(c.Secret)
}
