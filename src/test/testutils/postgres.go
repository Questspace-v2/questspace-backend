package testutils

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"

	"questspace/internal/pgdb/migrations"
)

const (
	TestPGUser     = "postgres"
	TestPGPassword = "postgres"
	TestPGDatabase = "questspace"
)

type DockerPG struct {
	pool     *dockertest.Pool
	resource *dockertest.Resource

	host string
	port string
}

func StartDockerPG() *DockerPG {
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not construct pool: %s", err)
	}

	err = pool.Client.Ping()
	if err != nil {
		log.Fatalf("Could not connect to Docker: %s", err)
	}

	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "16",
		Env: []string{
			"POSTGRES_PASSWORD=" + TestPGPassword,
			"POSTGRES_USER=" + TestPGUser,
			"POSTGRES_DB=" + TestPGDatabase,
		},
	}, func(config *docker.HostConfig) {
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}
	_ = resource.Expire(3600)

	hostAndPort := resource.GetHostPort("5432/tcp")
	split := strings.Split(hostAndPort, ":")
	host := split[0]
	port := split[1]
	databaseUrl := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable",
		TestPGUser, TestPGPassword, hostAndPort, TestPGDatabase,
	)

	var db *sql.DB
	if err = pool.Retry(func() error {
		db, err = sql.Open("pgx", databaseUrl)
		if err != nil {
			return err
		}
		timeoutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err = db.PingContext(timeoutCtx); err != nil {
			_ = db.Close()
			return err
		}
		return nil
	}); err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}
	runMigrations(db)
	_ = db.Close()

	return &DockerPG{pool: pool, resource: resource, host: host, port: port}
}

func (d *DockerPG) Close() {
	if err := d.pool.Purge(d.resource); err != nil {
		log.Fatalf("Could not purge resource: %s", err)
	}
}

func runMigrations(db *sql.DB) {
	migration, err := migrations.QuestspaceAsText()
	if err != nil {
		log.Fatalf("Could not load migrations: %s", err)
	}
	if _, err := db.Exec(migration); err != nil {
		log.Fatalf("Could not execute migrations: %v", err)
	}
}

func (d *DockerPG) Port() string {
	return d.port
}

func (d *DockerPG) Host() string {
	return d.host
}

func (d *DockerPG) Clean(t *testing.T) {
	hostAndPort := d.resource.GetHostPort("5432/tcp")
	databaseUrl := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable",
		TestPGUser, TestPGPassword, hostAndPort, TestPGDatabase,
	)
	var db *sql.DB
	if err := d.pool.Retry(func() error {
		var err error
		db, err = sql.Open("pgx", databaseUrl)
		if err != nil {
			return err
		}
		timeoutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err = db.PingContext(timeoutCtx); err != nil {
			_ = db.Close()
			return err
		}
		return nil
	}); err != nil {
		t.Errorf("Could not connect to docker: %s", err)
		t.FailNow()
	}
	if _, err := db.Exec("DROP SCHEMA questspace CASCADE"); err != nil {
		t.Errorf("Could not drop schema: %s", err)
		t.FailNow()
	}

	runMigrations(db)
	_ = db.Close()
}
