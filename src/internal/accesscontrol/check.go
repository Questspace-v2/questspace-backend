package accesscontrol

import (
	"context"
	"net/http"

	"questspace/pkg/httperrors"
	"questspace/pkg/storage"

	"github.com/yandex/perforator/library/go/core/xerrors"
)

func Check(ctx context.Context, s storage.AccessStorage, user *storage.User) error {
	hasAccess, err := s.HasAccess(ctx, user.ID)
	if err != nil {
		return xerrors.Errorf("check access: %w", err)
	}

	if !hasAccess {
		return httperrors.Errorf(http.StatusLocked, "user %s has no access", user.ID.String())
	}
	return nil
}
