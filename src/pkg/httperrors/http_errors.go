package httperrors

import (
	"golang.org/x/xerrors"
)

var _ error = &HTTPError{}

type HTTPError struct {
	Code int
	err  error
}

func New(httpCode int, msg string) *HTTPError {
	return &HTTPError{Code: httpCode, err: xerrors.New(msg)}
}

func Errorf(httpCode int, tmpl string, args ...interface{}) *HTTPError {
	return &HTTPError{Code: httpCode, err: xerrors.Errorf(tmpl, args...)}
}

func WrapWithCode(httpCode int, err error) error {
	return &HTTPError{Code: httpCode, err: xerrors.Errorf("%w", err)}
}

func (e *HTTPError) Error() string {
	return e.err.Error()
}

func (e *HTTPError) Unwrap() error {
	return e.err
}
