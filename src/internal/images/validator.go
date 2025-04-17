package images

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/yandex/perforator/library/go/core/xerrors"
	"golang.org/x/sync/errgroup"

	"questspace/pkg/httperrors"
)

type Config struct {
	MaxBodySize      int64         `yaml:"max-body-size"`
	MaxCompositeSize int64         `yaml:"max-composite-size"`
	Timeout          time.Duration `yaml:"timeout"`
}

type Validator struct {
	maxSize          int64
	maxCompositeSize int64
	client           *http.Client
	timeout          time.Duration
	mimeTypePrefixes []string
}

var (
	defaultPrefixes = []string{"image/"}
)

type validatorParams struct {
	MIMETypePrefixes []string
}

type Option func(p *validatorParams)

func WithMIMETypePrefixes(prefixes ...string) Option {
	return func(p *validatorParams) {
		p.MIMETypePrefixes = prefixes
	}
}

func NewValidator(httpClient *http.Client, config *Config, opts ...Option) Validator {
	params := validatorParams{
		MIMETypePrefixes: defaultPrefixes,
	}
	for _, opt := range opts {
		opt(&params)
	}

	return Validator{
		maxSize:          config.MaxBodySize,
		maxCompositeSize: config.MaxCompositeSize,
		client:           httpClient,
		timeout:          config.Timeout,
		mimeTypePrefixes: params.MIMETypePrefixes,
	}
}

func (v *Validator) ValidateImageURL(ctx context.Context, imageURL string) error {
	if v.timeout != 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, v.timeout)
		defer cancel()
	}

	size, err := v.validateImage(ctx, imageURL)
	if err != nil {
		return err
	}

	if size > v.maxSize {
		return httperrors.Errorf(
			http.StatusRequestEntityTooLarge,
			"image too large: %s vs max allowed %s",
			formatSize(size),
			formatSize(v.maxSize),
		)
	}

	return nil
}

func (v *Validator) ValidateImageURLs(ctx context.Context, imageURLs ...string) error {
	if v.timeout != 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, v.timeout)
		defer cancel()
	}
	var compositeSize atomic.Int64

	sema := make(chan struct{}, 20)
	errgrp, ctx := errgroup.WithContext(ctx)
	for _, url := range imageURLs {
		errgrp.Go(func() error {
			select {
			case <-ctx.Done():
				return httperrors.Errorf(http.StatusGatewayTimeout, "validate image timeout: %w", ctx.Err())
			case sema <- struct{}{}:
			}
			defer func() { <-sema }()

			size, validateErr := v.validateImage(ctx, url)
			if validateErr != nil {
				return xerrors.Errorf("validate %q: %w", url, validateErr)
			}
			compositeSize.Add(size)
			return nil
		})
	}
	if err := errgrp.Wait(); err != nil {
		return err
	}
	if compositeSize.Load() > v.maxCompositeSize {
		return httperrors.Errorf(
			http.StatusRequestEntityTooLarge,
			"total size of images is too large: %s vs max allowed %s",
			formatSize(compositeSize.Load()),
			formatSize(v.maxSize),
		)
	}

	return nil
}

func (v *Validator) validateImage(ctx context.Context, imageURL string) (size int64, err error) {
	if imageURL == "" {
		return 0, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, http.NoBody)
	if err != nil {
		return 0, httperrors.Errorf(http.StatusBadRequest, "bad url %q: %w", imageURL, err)
	}
	resp, err := v.client.Do(req)
	if err != nil {
		return 0, xerrors.Errorf("do request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	contentLength, err := io.Copy(io.Discard, resp.Body)
	if err != nil {
		return 0, xerrors.Errorf("read body: %w", err)
	}

	contentType := resp.Header.Get("Content-Type")
	if !v.suitsPrefixes(contentType) {
		return 0, httperrors.Errorf(http.StatusUnsupportedMediaType, "unsupported Content-Type: %q", contentType)
	}
	return contentLength, nil
}

func (v *Validator) suitsPrefixes(mimeType string) bool {
	for _, prefix := range v.mimeTypePrefixes {
		if strings.HasPrefix(mimeType, prefix) {
			return true
		}
	}
	return false
}

func formatSize(size int64) string {
	sizeKiB := size / 1024
	if sizeKiB < 1024 {
		return strconv.FormatInt(sizeKiB, 10) + " KiB"
	}
	sizeMiB := sizeKiB / 1024
	return strconv.FormatInt(sizeMiB, 10) + " MiB"
}
