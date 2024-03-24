package teams

import (
	"database/sql"
	"net/http"

	"questspace/pkg/application/httperrors"

	"questspace/pkg/dbnode"

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

// HandleGet handles GET /teams/:id request
//
//		@Summary	Get team by id
//		@Param		team_id	path		string						true	"Team id"
//		@Success	200		{object}	storage.Team
//	    @Failure    400
//	    @Failure    404
//		@Router		/teams/{team_id} [get]
func (h *Handler) HandleGet(c *gin.Context) error {
	teamID := c.Param("id")

	s, err := h.factory.NewStorage(c, dbnode.Alive)
	if err != nil {
		return xerrors.Errorf("get storage: %w", err)
	}
	teamService := teams.NewService(s, h.inviteLinkPrefix)
	team, err := teamService.GetTeam(c, teamID)
	if err != nil {
		return xerrors.Errorf("get team %q: %w", teamID, err)
	}

	c.JSON(http.StatusOK, team)
	return nil
}

// HandleGetMany handles GET /quest/:id/teams request
//
//		@Summary	Get all teams by quest id
//		@Param		quest_id	path		string						true	"Quest id"
//		@Success	200		{object}	[]storage.Team
//	    @Failure    400
//		@Router		/quest/{quest_id}/teams [get]
func (h *Handler) HandleGetMany(c *gin.Context) error {
	questID := c.Param("id")

	s, err := h.factory.NewStorage(c, dbnode.Alive)
	if err != nil {
		return xerrors.Errorf("get storage: %w", err)
	}
	teamService := teams.NewService(s, h.inviteLinkPrefix)
	questTeams, err := teamService.GetQuestTeams(c, questID)
	if err != nil {
		return xerrors.Errorf("get teams of quest %q: %w", questID, err)
	}

	c.JSON(http.StatusOK, questTeams)
	return nil
}

type UpdateRequest struct {
	Name string `json:"name"`
}

// HandleUpdate handles POST /teams/:id request
//
//		@Summary	Change team information
//		@Param		team_id	path		string						true	"Team id"
//		@Param		request	body		UpdateRequest						true	"New information"
//		@Success	200		{object} storage.Team
//	    @Failure    400
//	    @Failure    403
//	    @Failure    404
//		@Router		/teams/{team_id} [post]
func (h *Handler) HandleUpdate(c *gin.Context) error {
	teamID := c.Param("id")
	req, err := transport.UnmarshalRequestData[UpdateRequest](c.Request)
	if err != nil {
		return xerrors.Errorf("%w", err)
	}
	uauth, err := jwt.GetUserFromContext(c)
	if err != nil {
		return xerrors.Errorf("%w", err)
	}

	s, err := h.factory.NewStorage(c, dbnode.Master)
	if err != nil {
		return xerrors.Errorf("get storage: %w", err)
	}
	teamService := teams.NewService(s, h.inviteLinkPrefix)
	team, err := teamService.UpdateTeamName(c, uauth, &storage.ChangeTeamNameRequest{ID: teamID, Name: req.Name})
	if err != nil {
		return xerrors.Errorf("update name: %w", err)
	}

	c.JSON(http.StatusOK, team)
	return nil
}

// HandleDelete handles DELETE /teams/:id request
//
//		@Summary	Delete team by id
//		@Param		team_id	path		string						true	"Team id"
//		@Success	200
//	    @Failure    400
//	    @Failure    403
//	    @Failure    404
//		@Router		/teams/{team_id} [delete]
func (h *Handler) HandleDelete(c *gin.Context) error {
	teamID := c.Param("id")
	uauth, err := jwt.GetUserFromContext(c)
	if err != nil {
		return xerrors.Errorf("%w", err)
	}

	s, err := h.factory.NewStorage(c, dbnode.Master)
	if err != nil {
		return xerrors.Errorf("get storage: %w", err)
	}
	teamService := teams.NewService(s, h.inviteLinkPrefix)
	if err := teamService.DeleteTeam(c, uauth, &storage.DeleteTeamRequest{ID: teamID}); err != nil {
		return xerrors.Errorf("delete team %q: %w", teamID, err)
	}

	c.Status(http.StatusOK)
	return nil
}

type ChangeLeaderRequest struct {
	NewCaptainID string `json:"new_captain_id"`
}

// HandleChangeLeader handles POST /teams/:id/captain request
//
//		@Summary	Change team captain
//		@Param		team_id	path		string						true	"Team id"
//		@Param		request	body		ChangeLeaderRequest						true	"Change captain request"
//		@Success	200		{object} storage.Team
//	    @Failure    400
//	    @Failure    403
//	    @Failure    404
//		@Router		/teams/{team_id}/captain [post]
func (h *Handler) HandleChangeLeader(c *gin.Context) error {
	teamID := c.Param("id")
	req, err := transport.UnmarshalRequestData[ChangeLeaderRequest](c.Request)
	if err != nil {
		return xerrors.Errorf("%w", err)
	}
	uauth, err := jwt.GetUserFromContext(c)
	if err != nil {
		return xerrors.Errorf("%w", err)
	}

	s, err := h.factory.NewStorage(c, dbnode.Master)
	if err != nil {
		return xerrors.Errorf("get storage: %w", err)
	}
	teamService := teams.NewService(s, h.inviteLinkPrefix)
	team, err := teamService.ChangeLeader(c, uauth, &storage.ChangeLeaderRequest{ID: teamID, CaptainID: req.NewCaptainID})
	if err != nil {
		return xerrors.Errorf("change captain of team %q to %q: %w", teamID, req.NewCaptainID, err)
	}

	c.JSON(http.StatusOK, team)
	return nil
}

// HandleLeave handles POST /teams/:id/leave request
//
//		@Summary	Leave the team
//		@Param		team_id	path		string						true	"Team id"
//		@Param		new_captain_id	query		string						false	"New captain (if leader leaves)"
//		@Success	200		{object} storage.Team
//	    @Failure    400
//	    @Failure    403
//	    @Failure    404
//		@Router		/teams/{team_id}/leave [post]
func (h *Handler) HandleLeave(c *gin.Context) error {
	teamID := c.Param("id")
	newCaptainID := c.Query("new_captain")
	uauth, err := jwt.GetUserFromContext(c)
	if err != nil {
		return xerrors.Errorf("%w", err)
	}

	s, tx, err := h.factory.NewStorageTx(c, &sql.TxOptions{Isolation: sql.LevelWriteCommitted})
	if err != nil {
		return xerrors.Errorf("start tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	teamService := teams.NewService(s, h.inviteLinkPrefix)
	team, err := teamService.LeaveTeam(c, uauth, teamID, newCaptainID)
	if err != nil {
		return xerrors.Errorf("leave team: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return xerrors.Errorf("commit tx: %w", err)
	}

	c.JSON(http.StatusOK, team)
	return nil
}

// HandleRemoveUser handles DELETE /teams/:id/:user_id request
//
//		@Summary	Remove member from team
//		@Param		team_id	path		string						true	"Team id"
//		@Param		member_id	path		string						true	"Member id"
//		@Success	200		{object} storage.Team
//	    @Failure    400
//	    @Failure    403
//	    @Failure    404
//		@Router		/teams/{team_id}/{member_id} [delete]
func (h *Handler) HandleRemoveUser(c *gin.Context) error {
	teamID := c.Param("id")
	userID := c.Param("user_id")
	if userID == "" {
		return httperrors.New(http.StatusBadRequest, "no users to delete")
	}
	uauth, err := jwt.GetUserFromContext(c)
	if err != nil {
		return xerrors.Errorf("%w", err)
	}

	s, err := h.factory.NewStorage(c, dbnode.Master)
	if err != nil {
		return xerrors.Errorf("get storage: %w", err)
	}
	teamService := teams.NewService(s, h.inviteLinkPrefix)
	team, err := teamService.RemoveUser(c, uauth, &storage.RemoveUserRequest{UserID: userID, ID: teamID})
	if err != nil {
		return xerrors.Errorf("remove user %q from team %q: %w", userID, teamID, err)
	}

	c.JSON(http.StatusOK, team)
	return nil
}
