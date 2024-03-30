package pgconfig

import (
	"database/sql"
	"fmt"
	"os"
	"path"

	"questspace/pkg/secret"

	_ "github.com/jackc/pgx/v5/stdlib"
	"golang.org/x/xerrors"
	"golang.yandex/hasql"
	"gopkg.in/yaml.v3"
)

var defaultCrtPath = path.Join(os.Getenv("HOME"), ".postgresql", "root.crt")

type Config struct {
	Hosts       []string   `yaml:"hosts"`
	Port        uint16     `yaml:"port"`
	Database    string     `yaml:"database"`
	User        secret.Ref `yaml:"user"`
	Password    secret.Ref `yaml:"password"`
	SSLMode     string     `yaml:"sslmode,omitempty"`
	SSLRootCert string     `yaml:"sslrootcert,omitempty"`
}

func (c *Config) Validate() error {
	if len(c.Hosts) == 0 {
		return xerrors.New("no database hosts were provided")
	}
	return nil
}

func (c *Config) GetNodes() ([]hasql.Node, []error) {
	if err := c.Validate(); err != nil {
		return nil, []error{xerrors.Errorf("validate config: %w", err)}
	}

	var errs []error
	user, userErr := c.User.Read()
	if userErr != nil {
		errs = append(errs, xerrors.Errorf("read user secret: %w", userErr))
	}
	pw, pwErr := c.Password.Read()
	if pwErr != nil {
		errs = append(errs, xerrors.Errorf("read password secret: %w", pwErr))
	}
	if len(errs) > 0 {
		return nil, errs
	}

	nodes := make([]hasql.Node, 0, len(c.Hosts))
	for _, host := range c.Hosts {
		db, err := sql.Open("pgx", c.getDSN(host, user, pw))
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

func (c *Config) getDSN(host, user, password string) string {
	return fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=%s sslrootcert=%s target_session_attrs=read-write",
		host, c.Port, c.Database, user, password, c.SSLMode, c.SSLRootCert)
}

func (c *Config) UnmarshalYAML(value *yaml.Node) error {
	type configAlias Config
	temp := &configAlias{}

	if err := value.Decode(temp); err != nil {
		return err
	}

	if temp.SSLMode == "" {
		temp.SSLMode = "disable"
	}
	if temp.SSLRootCert == "" {
		temp.SSLRootCert = defaultCrtPath
	}

	*c = Config(*temp)
	return nil
}
