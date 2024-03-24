package pgdb

import (
	"testing"

	"github.com/stretchr/testify/require"

	"questspace/internal/pgdb/pgtest"
)

func TestNewClient(t *testing.T) {
	db := pgtest.NewEmbeddedQuestspaceDB(t)
	client := NewClient(db)
	require.NotNil(t, client)
}
