package transport

import (
	"encoding/json"
	"io"
	"net/http"

	"questspace/pkg/application/httperrors"
)

func UnmarshalRequestData[T any](req *http.Request) (*T, error) {
	defer func() { _ = req.Body.Close() }()
	data, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, httperrors.Errorf(http.StatusBadRequest, "read body: %w", err)
	}
	unmarshalled := new(T)
	if err := json.Unmarshal(data, unmarshalled); err != nil {
		return nil, httperrors.Errorf(http.StatusBadRequest, "unmarshal request: %w", err)
	}
	return unmarshalled, nil
}
