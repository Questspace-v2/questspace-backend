package pgdb

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"sync/atomic"
	"testing"

	sq "github.com/Masterminds/squirrel"
	"github.com/stretchr/testify/require"

	"questspace/pkg/dbnode"
	"questspace/pkg/storage"
)

type QuestspaceClientFactory interface {
	NewStorage(context.Context, dbnode.PickCriteria) (storage.QuestSpaceStorage, error)
	NewStorageTx(context.Context, *sql.TxOptions) (storage.QuestSpaceStorage, driver.Tx, error)
}

type mainFactory struct {
	picker dbnode.Picker
}

func NewQuestspaceClientFactory(p dbnode.Picker) QuestspaceClientFactory {
	return &mainFactory{picker: p}
}

func (m *mainFactory) NewStorage(ctx context.Context, cr dbnode.PickCriteria) (storage.QuestSpaceStorage, error) {
	var db *sql.DB
	var err error

	switch cr {
	case dbnode.Alive:
		db, err = m.picker.AliveNode(ctx)
	case dbnode.Master:
		db, err = m.picker.MasterNode(ctx)
	}
	if err != nil {
		return nil, err
	}

	return NewClientM(sq.WrapStdSqlCtx(db)), err
}

func (m *mainFactory) NewStorageTx(ctx context.Context, options *sql.TxOptions) (storage.QuestSpaceStorage, driver.Tx, error) {
	tx, err := m.picker.MasterNodeTx(ctx, options)
	if err != nil {
		return nil, nil, err
	}
	return NewClientM(sq.WrapStdSqlCtx(tx)), tx, nil
}

var _ QuestspaceClientFactory = &FakeClientFactory{}

type FakeClientFactory struct {
	s     storage.QuestSpaceStorage
	cmCnt atomic.Uint64
}

func NewFakeClientFactory(s storage.QuestSpaceStorage) *FakeClientFactory {
	return &FakeClientFactory{s: s}
}

func (f *FakeClientFactory) NewStorage(_ context.Context, _ dbnode.PickCriteria) (storage.QuestSpaceStorage, error) {
	return f.s, nil
}

func (f *FakeClientFactory) NewStorageTx(_ context.Context, _ *sql.TxOptions) (storage.QuestSpaceStorage, driver.Tx, error) {
	return f.s, f, nil
}

func (f *FakeClientFactory) Commit() error {
	f.cmCnt.Add(1)
	return nil
}

func (f *FakeClientFactory) Rollback() error {
	return nil
}

func (f *FakeClientFactory) ExpectCommit(t *testing.T) {
	cnt := f.cmCnt.Load()
	require.NotZero(t, cnt, "Expected transaction commit")
	require.Equal(t, uint64(1), cnt, "Too many transaction commits")
}

func (f *FakeClientFactory) ExpectCommits(t *testing.T, expectedCnt uint64) {
	cnt := f.cmCnt.Load()
	require.Equal(t, expectedCnt, cnt, "Expected %d commits", expectedCnt)
}