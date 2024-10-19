package testhandlers

import (
	"context"
	"net/http"
	"time"

	"questspace/internal/qtime"
	"questspace/pkg/httperrors"
	"questspace/pkg/transport"
)

const defaultDuration = time.Hour * 24

func HandleWait(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	if !qtime.IsTestTimeMode() {
		return httperrors.New(http.StatusForbidden, "cannot wait in production environment")
	}
	waitTime := defaultDuration
	if dString := transport.Query(r, "d"); len(dString) > 0 {
		var err error
		waitTime, err = time.ParseDuration(dString)
		if err != nil {
			return httperrors.Errorf(http.StatusBadRequest, "bad duration: %w", err)
		}
	}
	qtime.Wait(waitTime)

	w.WriteHeader(http.StatusOK)
	return nil
}
