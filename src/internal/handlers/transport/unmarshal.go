package transport

import (
	"encoding/json"
	"net/http"

	"questspace/pkg/application/httperrors"
)

func UnmarshalRequestData[T any](req *http.Request) (*T, error) {
	defer func() { _ = req.Body.Close() }()
	unmarshalled := new(T)
	if err := json.NewDecoder(req.Body).Decode(unmarshalled); err != nil {
		return nil, httperrors.Errorf(http.StatusBadRequest, "unmarshal request: %w", err)
	}
	return unmarshalled, nil
}
