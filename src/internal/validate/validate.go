package validate

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/xerrors"

	"questspace/pkg/httperrors"
)

const imgHeadTimeout = time.Second * 5

func ImageURL(ctx context.Context, client http.Client, imgUrl string) error {
	if imgUrl == "" {
		return nil
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, imgHeadTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(timeoutCtx, http.MethodHead, imgUrl, http.NoBody)
	if err != nil {
		return xerrors.Errorf("create http request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return xerrors.Errorf("get imgUrl head: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, resp.Body)

	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		return httperrors.Errorf(http.StatusUnsupportedMediaType, "non-image Content-Type: %q", contentType)
	}
	return nil
}
