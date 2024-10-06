package transport

import (
	"net/http"

	"github.com/gofrs/uuid"
	"github.com/julienschmidt/httprouter"

	"questspace/pkg/httperrors"
	"questspace/pkg/storage"
)

func StringParam(r *http.Request, key string) (string, bool) {
	ctx := r.Context()
	params := httprouter.ParamsFromContext(ctx)
	val := params.ByName(key)
	return val, len(val) == 0
}

func UUIDParam(r *http.Request, key string) (storage.ID, error) {
	// TODO(svayp11): Return UUID
	ctx := r.Context()
	params := httprouter.ParamsFromContext(ctx)
	stringID := params.ByName(key)
	if len(stringID) == 0 {
		return "", storage.ErrNotFound
	}
	var id uuid.UUID
	if err := id.UnmarshalText([]byte(stringID)); err != nil {
		return "", httperrors.Errorf(http.StatusBadRequest, "invalid uuid: %w", err)
	}
	return storage.ID(stringID), nil
}

func QueryArray(r *http.Request, key string) []string {
	return r.URL.Query()[key]
}

func Query(r *http.Request, key string) string {
	return r.URL.Query().Get(key)
}
