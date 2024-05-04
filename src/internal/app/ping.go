package app

import (
	"net/http"

	"questspace/pkg/transport"
)

func Ping(w http.ResponseWriter, _ *http.Request) {
	transport.ServeText(w, http.StatusOK, "pong")
}
