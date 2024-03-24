package pgtest

import (
	"database/sql/driver"
	"os"
	"testing"

	sq "github.com/Masterminds/squirrel"
	"github.com/stretchr/testify/require"

	"questspace/internal/pgdb/migrations"
	"questspace/pkg/embedpg"
)

func NewEmbeddedQuestspaceDB(t *testing.T) sq.RunnerContext {
	//TODO(svayp11): find workaround to run tests in CI
	if os.Getenv("CI") == "true" {
		t.Skipf("running in ci, cannot download PG, skipping...")
	}

	db := embedpg.NewEmbeddedPGDB(t)
	migrationText, err := migrations.QuestspaceAsText()
	require.NoError(t, err)
	_, err = db.Exec(migrationText)
	require.NoError(t, err)
	return sq.WrapStdSqlCtx(db)
}

func NewEmbeddedQuestspaceTx(t *testing.T) (sq.RunnerContext, driver.Tx) {
	//TODO(svayp11): find workaround to run tests in CI
	if os.Getenv("CI") == "true" {
		t.Skipf("running in ci, cannot download PG, skipping...")
	}

	db := embedpg.NewEmbeddedPGDB(t)
	migrationText, err := migrations.QuestspaceAsText()
	require.NoError(t, err)
	_, err = db.Exec(migrationText)
	require.NoError(t, err)
	tx, err := db.Begin()
	require.NoError(t, err)

	t.Cleanup(func() { _ = tx.Rollback() })
	return sq.WrapStdSqlCtx(tx), tx
}
