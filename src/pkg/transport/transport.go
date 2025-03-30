package transport

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/yandex/perforator/library/go/core/xerrors"

	"questspace/pkg/httperrors"
)

func UnmarshalRequestData[T any](req *http.Request) (T, error) {
	defer func() { _ = req.Body.Close() }()
	var unmarshalled T
	if err := json.NewDecoder(req.Body).Decode(&unmarshalled); err != nil {
		return unmarshalled, httperrors.Errorf(http.StatusBadRequest, "unmarshal request: %w", err)
	}
	return unmarshalled, nil
}

func ServeJSONResponse[T any](w http.ResponseWriter, status int, response T) error {
	data, err := json.Marshal(response)
	if err != nil {
		return xerrors.Errorf("marshal json data: %w", err)
	}
	if h := w.Header().Get("Content-Type"); len(h) == 0 {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
	}
	w.WriteHeader(status)
	_, _ = w.Write(data)
	return nil
}

func ServeText(w http.ResponseWriter, status int, text string) {
	if h := w.Header().Get("Content-Type"); len(h) == 0 {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	}
	w.WriteHeader(status)
	_, _ = fmt.Fprint(w, text)
}

func ServeTextf(w http.ResponseWriter, status int, format string, a ...any) {
	if h := w.Header().Get("Content-Type"); len(h) == 0 {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	}
	w.WriteHeader(status)
	_, _ = fmt.Fprintf(w, format, a...)
}

func ServeErr(w http.ResponseWriter, status int, err error) {
	ServeText(w, status, err.Error())
}
