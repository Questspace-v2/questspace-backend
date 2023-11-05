package dbconfig

import (
	"fmt"
	"os"
	"path"

	"gopkg.in/yaml.v3"

	"questspace/pkg/application"
)

var defaultCrtPath = path.Join(os.Getenv("HOME"), ".postgresql", "root.crt")

type Config struct {
	Host        string `yaml:"host"`
	Port        uint16 `yaml:"port"`
	Database    string `yaml:"database"`
	User        string `yaml:"user"`
	Password    string `yaml:"password"`
	SSLMode     string `yaml:"sslmode,omitempty"`
	SSLRootCert string `yaml:"sslrootcert,omitempty"`
}

func (c *Config) GetDSN() string {
	if c.SSLMode == "" {
		c.SSLMode = "disable"
	}
	if c.SSLRootCert == "" {
		c.SSLRootCert = defaultCrtPath
	}
	return fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=%s sslrootcert=%s target_session_attrs=read-write",
		c.Host, c.Port, c.Database, c.User, c.Password, c.SSLMode, c.SSLRootCert)
}

func (c *Config) UnmarshalYAML(value *yaml.Node) error {
	type configAlias Config
	temp := &configAlias{}

	if err := value.Decode(temp); err != nil {
		return err
	}

	unSec, err := application.ReadSecret(temp.User)
	if err != nil {
		return err
	}
	temp.User = unSec

	pwSec, err := application.ReadSecret(temp.Password)
	if err != nil {
		return err
	}
	temp.Password = pwSec

	*c = Config(*temp)
	return nil
}
