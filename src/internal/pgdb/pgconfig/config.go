package pgconfig

import (
	"database/sql"
	"fmt"
	"os"
	"path"

	_ "github.com/jackc/pgx/stdlib"
	"golang.org/x/xerrors"
	"golang.yandex/hasql"
	"gopkg.in/yaml.v3"

	"questspace/pkg/application"
)

var defaultCrtPath = path.Join(os.Getenv("HOME"), ".postgresql", "root.crt")

type Config struct {
	Hosts       []string `yaml:"hosts"`
	Port        uint16   `yaml:"port"`
	Database    string   `yaml:"database"`
	User        string   `yaml:"user"`
	Password    string   `yaml:"password"`
	SSLMode     string   `yaml:"sslmode,omitempty"`
	SSLRootCert string   `yaml:"sslrootcert,omitempty"`
}

func (c *Config) GetNodes() ([]hasql.Node, []error) {
	nodes := make([]hasql.Node, 0, len(c.Hosts))
	var errs []error

	for _, host := range c.Hosts {
		db, err := sql.Open("pgx", c.getDSN(host))
		if err != nil {
			errs = append(errs, err)
			continue
		}
		nodes = append(nodes, hasql.NewNode(host, db))
	}

	if len(errs) > 0 {
		return nil, errs
	}
	return nodes, nil
}

func (c *Config) getDSN(host string) string {
	return fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=%s sslrootcert=%s target_session_attrs=read-write",
		host, c.Port, c.Database, c.User, c.Password, c.SSLMode, c.SSLRootCert)
}

func (c *Config) UnmarshalYAML(value *yaml.Node) error {
	type configAlias Config
	temp := &configAlias{}

	if err := value.Decode(temp); err != nil {
		return err
	}

	if len(temp.Hosts) == 0 {
		return xerrors.New("no database hosts were provided")
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

	if temp.SSLMode == "" {
		temp.SSLMode = "disable"
	}
	if temp.SSLRootCert == "" {
		temp.SSLRootCert = defaultCrtPath
	}

	*c = Config(*temp)
	return nil
}
