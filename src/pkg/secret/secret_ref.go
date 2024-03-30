package secret

import (
	"encoding/json"
	"os"
	"strings"
	"sync"

	"golang.org/x/xerrors"
	"gopkg.in/yaml.v3"
)

// Ref is a holder for critical secret data.
// The main purpose of ref is to hold and read config secrets without directly stating them.
// Secrets may be read from files (default) or from environment (if "env:" prefix is present)
type Ref struct {
	ref  string
	once *sync.Once
	val  string
}

func NewRef(refString string) *Ref {
	return &Ref{
		ref:  refString,
		once: &sync.Once{},
	}
}

func NewEnvRef(envKey string) *Ref {
	return NewRef("env:" + envKey)
}

func (r *Ref) UnmarshalYAML(value *yaml.Node) error {
	var val string
	if err := value.Decode(&val); err != nil {
		return err
	}

	r.ref = val
	r.once = &sync.Once{}
	return nil
}

func (r *Ref) UnmarshalJSON(data []byte) error {
	var val string

	if err := json.Unmarshal(data, &val); err != nil {
		return err
	}

	r.ref = val
	r.once = &sync.Once{}
	return nil
}

func (r *Ref) Read() (string, error) {
	var err error
	r.once.Do(func() {
		r.val, err = r.load()
	})
	if err != nil {
		return "", xerrors.Errorf("load secret: %w", err)
	}

	return r.val, nil
}

func (r *Ref) load() (string, error) {
	value := r.ref
	if strings.HasPrefix(value, "env:") {
		secretRef, ok := os.LookupEnv(strings.SplitN(value, ":", 2)[1])
		if ok {
			return secretRef, nil
		}
	}
	secret, err := os.ReadFile(value)
	if err != nil {
		return "", xerrors.Errorf("read secret file: %w", err)
	}
	return string(secret), nil
}
