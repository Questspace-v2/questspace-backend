package taskgroups

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/xerrors"

	"questspace/internal/handlers/transport"
	pgdb "questspace/internal/pgdb/client"
	"questspace/internal/questspace/taskgroups"
	"questspace/pkg/storage"
)

type Handler struct {
	clientFactory pgdb.QuestspaceClientFactory
}

func NewHandler(c pgdb.QuestspaceClientFactory) *Handler {
	return &Handler{clientFactory: c}
}

type TaskGroups []*storage.TaskGroup

// HandleBulkUpdate handles PATCH quest/:id/task-groups/bulk request
//
//		@Summary	Patch task groups by creating new ones, delete, update and reorder all ones. Returns all exising task groups.
//		@Param		request	body		storage.TaskGroupsBulkUpdateRequest	true	"Requests to delete/create/update task groups"
//		@Success	200		{object}	TaskGroups
//		@Failure	400
//	    @Failure    401
//		@Failure    403
//		@Router		/quest/{id}/task-groups/bulk [patch]
func (h *Handler) HandleBulkUpdate(c *gin.Context) error {
	//TODO(svayp11): add auth
	questID := c.Param("id")
	req, err := transport.UnmarshalRequestData[storage.TaskGroupsBulkUpdateRequest](c.Request)
	if err != nil {
		return xerrors.Errorf("%w", err)
	}
	req.QuestID = questID

	s, tx, err := h.clientFactory.NewStorageTx(c, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return xerrors.Errorf("get storage client: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	updater := taskgroups.NewUpdater(s)
	tasksGroups, err := updater.BulkUpdateTaskGroups(c, req)
	if err != nil {
		return xerrors.Errorf("bulk update: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return xerrors.Errorf("commit tx: %w", err)
	}

	c.JSON(http.StatusOK, tasksGroups)
	return nil
}
