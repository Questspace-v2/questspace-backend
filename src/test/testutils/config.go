package testutils

import (
	"bytes"
	"errors"
	"log"
	"os"
	"path/filepath"
	"text/template"

	"questspace/pkg/environment"
)

const (
	ConfigFileName = "dev.yaml"

	TestJWTSecret    = "secret"
	TestGoogleSecret = "google_secret"
	InviteLinkPrefix = ""

	postgresPasswordEnvVar = "POSTGRES_PASSWORD"
	postgresUserEnvVar     = "POSTGRES_USER"
	jwtSecretEnvVar        = "JWT_SECRET"
	googleSecretEnvVar     = "GOOGLE_SECRET"
)

type CloserFunc func()

const configTmpl = `
db:
  hosts:
    - {{ .DBHost }}
  port: {{ .DBPort }}
  database: {{ .DBName }}
  user: env:{{ .DBUserKey }}
  password: env:{{ .DBPasswordKey }}
  sslmode: disable

cors:
  allow-origins:
    - "*"
  allow-headers:
    - Authorization
  allow-methods: [GET, POST, PUT, PATCH, DELETE, OPTIONS, HEAD]

google-oauth:
  client-id: some_client_id
  client-secret: env:{{ .GoogleSecretKey }}

hash-cost: 1

jwt:
  secret: env:{{ .JWTSecretKey }}

teams:
  invite-link-prefix: {{ .InviteLinkPrefix }}

validator:
  timeout: 60s
  max-body-size: 5242880  # 5 MiB
`

type configTmplParams struct {
	DBHost           string
	DBPort           string
	DBName           string
	DBUserKey        string
	DBPasswordKey    string
	GoogleSecretKey  string
	JWTSecretKey     string
	InviteLinkPrefix string
}

func CreateTestingConfig() (path string, closer CloserFunc) {
	tempdir, err := os.MkdirTemp("", "configs-*")
	if err != nil {
		log.Fatalf("Failed to create temporary directory: %v", err)
	}

	tmpl := template.New("appconfig_tmpl")
	params := configTmplParams{
		DBHost:           PG.Host(),
		DBPort:           PG.Port(),
		DBName:           TestPGDatabase,
		DBUserKey:        postgresUserEnvVar,
		DBPasswordKey:    postgresPasswordEnvVar,
		GoogleSecretKey:  googleSecretEnvVar,
		JWTSecretKey:     jwtSecretEnvVar,
		InviteLinkPrefix: InviteLinkPrefix,
	}
	tmpl = template.Must(tmpl.Parse(configTmpl))

	buf := new(bytes.Buffer)
	if err := tmpl.Execute(buf, params); err != nil {
		log.Fatal(err)
	}

	configPath := filepath.Join(tempdir, ConfigFileName)
	configFile, err := os.Create(configPath)
	if err != nil {
		log.Fatal(err)
	}
	_, err = configFile.Write(buf.Bytes())
	if err != nil {
		log.Fatal(err)
	}

	if err := setenv(); err != nil {
		log.Fatal(err)
	}

	return tempdir, unsetenv
}

func setenv() error {
	var errs []error
	errs = append(errs,
		os.Setenv(postgresUserEnvVar, TestPGUser),
		os.Setenv(postgresPasswordEnvVar, TestPGPassword),
		os.Setenv(jwtSecretEnvVar, TestJWTSecret),
		os.Setenv(googleSecretEnvVar, TestGoogleSecret),
		os.Setenv(environment.AppEnvironmentEnvKey, "dev"),
	)
	return errors.Join(errs...)
}

func unsetenv() {
	_ = os.Unsetenv(postgresUserEnvVar)
	_ = os.Unsetenv(postgresPasswordEnvVar)
	_ = os.Unsetenv(jwtSecretEnvVar)
	_ = os.Unsetenv(googleSecretEnvVar)
	_ = os.Unsetenv(environment.AppEnvironmentEnvKey)
}
