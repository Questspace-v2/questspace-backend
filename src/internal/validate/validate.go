package validate

import (
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	aerrors "questspace/pkg/application/errors"

	"golang.org/x/xerrors"
)

func ImageURL(client http.Client, imgUrl string) error {
	if imgUrl == "" {
		return nil
	}

	u, err := url.Parse(imgUrl)
	if err != nil {
		return xerrors.Errorf("failed to parse url: %w", err)
	}
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(u.Path), "."))
	switch ext {
	case "jpg", "jpeg", "png", "gif", "bmp", "svg":
		return nil
	}

	resp, err := client.Head(imgUrl)
	if err != nil {
		return xerrors.Errorf("failed to get imgUrl head: %w", err)
	}
	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		return xerrors.Errorf("non-image Content-Type %s: %w", contentType, aerrors.ErrValidation)
	}
	return nil
}
