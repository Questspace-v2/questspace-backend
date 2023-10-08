package user

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"path/filepath"
	aerrors "questspace/pkg/application/errors"
	"questspace/pkg/storage"
	"strings"

	"golang.org/x/xerrors"

	"github.com/gin-gonic/gin"
)

const defaultAvatarURL = "https://api.dicebear.com/7.x/thumbs/svg"

type CreateHandler struct {
	storage storage.UserStorage
	fetcher http.Client
}

func NewCreateHandler(s storage.UserStorage, f http.Client) CreateHandler {
	return CreateHandler{
		storage: s,
		fetcher: f,
	}
}

func (h CreateHandler) Handle(c *gin.Context) error {
	data, err := c.GetRawData()
	if err != nil {
		return xerrors.Errorf("failed to get raw data: %w", err)
	}
	req := storage.CreateUserRequest{}
	if err := json.Unmarshal(data, &req); err != nil {
		return xerrors.Errorf("failed to unmarshall request: %w", err)
	}
	if err := h.validateImageUrl(req.AvatarURL); err != nil {
		return xerrors.Errorf("failed to validate image: %w", err)
	}
	if req.AvatarURL == "" {
		req.AvatarURL = defaultAvatarURL
	}
	user, err := h.storage.CreateUser(c, &req)
	if err != nil {
		if errors.Is(err, storage.ErrExists) {
			return aerrors.ErrBadRequest
		}
		return xerrors.Errorf("failed to create user: %w", err)
	}
	c.JSON(http.StatusOK, user)
	return nil
}

func (h CreateHandler) validateImageUrl(imgUrl string) error {
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

	resp, err := h.fetcher.Head(imgUrl)
	if err != nil {
		return xerrors.Errorf("failed to get imgUrl head: %w", err)
	}
	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		return xerrors.Errorf("non-image Content-Type %s: %w", contentType, aerrors.ErrValidation)
	}
	return nil
}
