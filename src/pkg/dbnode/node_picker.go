package dbnode

import (
	"context"
	"database/sql"
	"time"

	"golang.org/x/xerrors"
	"golang.yandex/hasql"
)

type PickCriteria int

var (
	Alive  PickCriteria = 0
	Master PickCriteria = 1
)

//go:generate mockgen -source=node_picker.go -destination mocks/node_picker.go -package mocks
type Picker interface {
	AliveNode(context.Context) (*sql.DB, error)
	MasterNode(ctx context.Context) (*sql.DB, error)
	MasterNodeTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

var _ Picker = &BasicPicker{}

type BasicPicker struct {
	cluster *hasql.Cluster
}

func NewBasicPicker(c *hasql.Cluster) *BasicPicker {
	return &BasicPicker{cluster: c}
}

func (p *BasicPicker) AliveNode(ctx context.Context) (*sql.DB, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()
	node, err := p.cluster.WaitForAlive(timeoutCtx)
	if err != nil {
		return nil, xerrors.Errorf("cannot get alive node: %w", err)
	}
	return node.DB(), err
}

func (p *BasicPicker) MasterNode(ctx context.Context) (*sql.DB, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()
	node, err := p.cluster.WaitForPrimary(timeoutCtx)
	if err != nil {
		return nil, xerrors.Errorf("cannot get primary node: %w", err)
	}
	return node.DB(), err
}

func (p *BasicPicker) MasterNodeTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()
	node, err := p.cluster.WaitForPrimary(timeoutCtx)
	if err != nil {
		return nil, xerrors.Errorf("cannot get primary node: %w", err)
	}
	tx, err := node.DB().BeginTx(ctx, opts)
	if err != nil {
		return nil, xerrors.Errorf("cannot start tx: %w", err)
	}
	return tx, nil
}
