package jwt

import (
	"questspace/pkg/secret"
)

type Config struct {
	Secret secret.Ref `yaml:"secret"`
}
