package pgdb

import (
	sq "github.com/Masterminds/squirrel"

	"questspace/pkg/storage"
)

const (
	uniqueViolationCode        = "23505"
	triggerActionExceptionCode = "P0001"
)

type Client struct {
	runner sq.RunnerContext
}

var _ storage.QuestSpaceStorage = &Client{}

func NewClient(r sq.RunnerContext) *Client {
	return &Client{runner: r}
}
