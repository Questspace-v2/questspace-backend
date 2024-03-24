package embedpg

import (
	"database/sql"
	"fmt"
	"io"
	"net"
	"testing"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	defaultUser     = "testuser"
	defaultPassword = "testpassword"
	defaultDB       = "questspace"
)

func NewEmbeddedPGDB(t *testing.T) *sql.DB {
	freePort, err := getFreePort()
	require.NoError(t, err)
	pgConf := embeddedpostgres.DefaultConfig().
		Username(defaultUser).
		Password(defaultPassword).
		Database(defaultDB).
		Version(embeddedpostgres.V16).
		Port(freePort).
		Logger(io.Discard)

	pg := embeddedpostgres.NewDatabase(pgConf)
	err = pg.Start()
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, pg.Stop())
	})

	db, err := sql.Open("pgx", getConnString(freePort))
	require.NoError(t, err)
	return db
}

func getConnString(port uint32) string {
	return fmt.Sprintf(
		"host=localhost port=%d user=%s password=%s dbname=%s sslmode=disable",
		port, defaultUser, defaultPassword, defaultDB,
	)
}

func getFreePort() (uint32, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer func() { _ = l.Close() }()
	return uint32(l.Addr().(*net.TCPAddr).Port), nil
}
