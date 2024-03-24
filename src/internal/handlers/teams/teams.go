package teams

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/xerrors"

	"questspace/internal/handlers/transport"
	pgdb "questspace/internal/pgdb/client"
	"questspace/internal/questspace/teams"
	"questspace/pkg/auth/jwt"
	"questspace/pkg/storage"
)

type Handler struct {
	factory          pgdb.QuestspaceClientFactory
	inviteLinkPrefix string
}

func NewHandler(f pgdb.QuestspaceClientFactory, prefix string) *Handler {
	return &Handler{
		factory:          f,
		inviteLinkPrefix: prefix,
	}
}

type CreateRequest struct {
	Name string `json:"name"`
}

// HandleCreate handles POST /quest/:id/teams request
//
//		@Summary	Create new team
//		@Param		quest_id	path		string						true	"Quest ID"
//		@Param		request	body		CreateRequest	true	"Desired team information"
//		@Success	200		{object}	storage.Team
//		@Failure	400
//	    @Failure    401
//	    @Failure    406
//		@Router		/quest/{quest_id}/teams [post]
func (h *Handler) HandleCreate(c *gin.Context) error {
	req, err := transport.UnmarshalRequestData[CreateRequest](c.Request)
	if err != nil {
		return xerrors.Errorf("%w", err)
	}
	questId := c.Param("id")
	uauth, err := jwt.GetUserFromContext(c)
	if err != nil {
		return xerrors.Errorf("%w", err)
	}

	storageReq := storage.CreateTeamRequest{
		Creator: uauth,
		QuestID: questId,
		Name:    req.Name,
	}
	s, tx, err := h.factory.NewStorageTx(c, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return xerrors.Errorf("start tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	teamService := teams.NewService(s, h.inviteLinkPrefix)
	team, err := teamService.CreateTeam(c, &storageReq)
	if err != nil {
		return xerrors.Errorf("create team: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return xerrors.Errorf("commit tx: %w", err)
	}

	c.JSON(http.StatusOK, team)
	return nil
}

// HandleJoin handles GET /teams/join/:path request
//
//		@Summary	Join team
//		@Param		invite_path	path		string						true	"Team invite url param"
//		@Success	200		{object}	storage.Team
//	    @Failure    401
//	    @Failure    406
//		@Router		/teams/join/{invite_path} [get]
func (h *Handler) HandleJoin(c *gin.Context) error {
	invitePath := c.Param("path")
	uauth, err := jwt.GetUserFromContext(c)
	if err != nil {
		return xerrors.Errorf("%w", err)
	}

	s, tx, err := h.factory.NewStorageTx(c, nil)
	if err != nil {
		return xerrors.Errorf("start tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	teamService := teams.NewService(s, h.inviteLinkPrefix)
	team, err := teamService.JoinTeam(c, &storage.JoinTeamRequest{InvitePath: invitePath, User: uauth})
	if err != nil {
		return xerrors.Errorf("join team: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return xerrors.Errorf("commit tx: %w", err)
	}

	c.JSON(http.StatusOK, team)
	return nil
}
