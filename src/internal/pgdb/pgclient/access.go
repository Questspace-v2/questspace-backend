package pgclient

import (
	"context"
	"database/sql"

	"questspace/pkg/logging"
	"questspace/pkg/storage"

	"github.com/yandex/perforator/library/go/core/xerrors"
	"go.uber.org/zap"
)

func (c *Client) HasAccess(ctx context.Context, id storage.ID) (bool, error) {
	const getAccessQuery = `
	SELECT 1 FROM questspace.permissions
	WHERE user_id = $1;
	`

	row := c.runner.QueryRowContext(ctx, getAccessQuery, getAccessQuery)
	var has int
	if err := row.Scan(&has); err != nil {
		if xerrors.Is(err, sql.ErrNoRows) {
			logging.Warn(ctx, "user has no access", zap.Stringer("user_id", id))
			return false, nil
		}
		return false, xerrors.Errorf("scan row: %w", err)
	}

	return true, nil
}
