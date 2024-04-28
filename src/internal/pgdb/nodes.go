package pgdb

import (
	"database/sql"

	_ "github.com/jackc/pgx/v5/stdlib"
	"golang.org/x/xerrors"
	"golang.yandex/hasql"

	"questspace/internal/pgdb/pgconfig"
)

func GetNodes(c *pgconfig.Config) (nodes []hasql.Node, errs []error) {
	if err := c.Validate(); err != nil {
		return nil, []error{xerrors.Errorf("validate config: %w", err)}
	}
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

	nodes = make([]hasql.Node, 0, len(c.Hosts))
	for _, host := range c.Hosts {
		db, err := sql.Open("pgx", c.GetDSNForHost(host, user, pw))
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
