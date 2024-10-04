package googleservice

import "questspace/pkg/secret"

type Config struct {
	ClientID     string     `yaml:"client-id"`
	ClientSecret secret.Ref `yaml:"client-secret"`
}
