package validate

import (
	"context"
	"net/http"
	"strings"
	"time"

	"golang.org/x/xerrors"
)

func ImageURL(ctx context.Context, client http.Client, imgUrl string) error {
	if imgUrl == "" {
		return nil
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()
	req, err := http.NewRequestWithContext(timeoutCtx, http.MethodHead, imgUrl, http.NoBody)
	if err != nil {
		return xerrors.Errorf("failed to create http request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return xerrors.Errorf("failed to get imgUrl head: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		return xerrors.Errorf("non-image Content-Type %s", contentType)
	}
	return nil
}
