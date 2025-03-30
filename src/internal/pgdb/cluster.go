package pgdb

import (
	"context"
	"time"

	"github.com/yandex/perforator/library/go/core/xerrors"
	"golang.yandex/hasql"
	"golang.yandex/hasql/checkers"
)

func CreateCluster(ctx context.Context, nodes []hasql.Node) (*hasql.Cluster, error) {
	cl, err := hasql.NewCluster(nodes, checkers.PostgreSQL, hasql.WithNodePicker(hasql.PickNodeClosest()))
	if err != nil {
		return nil, xerrors.Errorf("create cluster: %w", err)
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	if _, err := cl.WaitForAlive(timeoutCtx); err != nil {
		return nil, xerrors.Errorf("connect to database cluster: %w", err)
	}
	return cl, nil
}
