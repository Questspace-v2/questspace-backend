package dbconfig

import (
	"fmt"

	"gopkg.in/yaml.v3"

	"questspace/pkg/application"
)

type Config struct {
	Host     string `yaml:"host"`
	Port     uint16 `yaml:"port"`
	Database string `yaml:"database"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	SSLMode  string `yaml:"ssl_mode,omitempty"`
}

func (c *Config) GetDSN() string {
	sslMode := c.SSLMode
	if c.SSLMode == "" {
		sslMode = "disable"
	}
	return fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=%s target_session_attrs=write",
		c.Host, c.Port, c.Database, c.User, c.Password, sslMode)
}

func (c *Config) UnmarshalYAML(value *yaml.Node) error {
	type configAlias Config
	temp := &configAlias{}

	if err := value.Decode(temp); err != nil {
		return err
	}

	secret, err := application.ReadSecret(temp.Password)
	if err != nil {
		return err
	}

	temp.Password = secret
	*c = Config(*temp)
	return nil
}
