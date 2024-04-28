package teams

import (
	"context"
	"database/sql"
	"errors"
	"net/http"

	"golang.org/x/xerrors"

	"questspace/internal/handlers/quest"
	"questspace/internal/pgdb"
	"questspace/internal/questspace/quests"
	"questspace/internal/questspace/teams"
	"questspace/pkg/auth/jwt"
	"questspace/pkg/dbnode"
	"questspace/pkg/httperrors"
	"questspace/pkg/storage"
	"questspace/pkg/transport"
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

// HandleCreate handles POST /quest/{id}/teams request
//
// @Summary		Create new team
// @Tags		Teams
// @Param		quest_id	path		string					true	"Quest ID"
// @Param		request		body		CreateRequest	true	"Desired team information"
// @Success		200			{object}	storage.Team
// @Failure		400
// @Failure    	401
// @Failure    	406
// @Router		/quest/{quest_id}/teams [post]
// @Security 	ApiKeyAuth
func (h *Handler) HandleCreate(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	req, err := transport.UnmarshalRequestData[CreateRequest](r)
	if err != nil {
		return xerrors.Errorf("%w", err)
	}
	questId, err := transport.UUIDParam(r, "id")
	if err != nil {
		return xerrors.Errorf("%w", err)
	}
	uauth, err := jwt.GetUserFromContext(ctx)
	if err != nil {
		return xerrors.Errorf("%w", err)
	}

	storageReq := storage.CreateTeamRequest{
		Creator: uauth,
		QuestID: questId,
		Name:    req.Name,
	}
	s, tx, err := h.factory.NewStorageTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return xerrors.Errorf("start tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	teamService := teams.NewService(s, h.inviteLinkPrefix)
	team, err := teamService.CreateTeam(ctx, &storageReq)
	if err != nil {
		return xerrors.Errorf("create team: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return xerrors.Errorf("commit tx: %w", err)
	}

	if err = transport.ServeJSONResponse(w, http.StatusOK, team); err != nil {
		return err
	}
	return nil
}

// HandleJoin handles GET /teams/join/{path} request
//
// @Summary		Join team
// @Tags		Teams
// @Param		invite_path	path		string	true	"Team invite url param"
// @Success		200			{object}	storage.Team
// @Failure    	401
// @Failure    	406
// @Router		/teams/join/{invite_path} [get]
// @Security 	ApiKeyAuth
func (h *Handler) HandleJoin(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	invitePath, _ := transport.StringParam(r, "path")
	uauth, err := jwt.GetUserFromContext(ctx)
	if err != nil {
		return xerrors.Errorf("%w", err)
	}

	s, tx, err := h.factory.NewStorageTx(ctx, nil)
	if err != nil {
		return xerrors.Errorf("start tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	teamService := teams.NewService(s, h.inviteLinkPrefix)
	team, err := teamService.JoinTeam(ctx, &storage.JoinTeamRequest{InvitePath: invitePath, User: uauth})
	if err != nil {
		return xerrors.Errorf("join team: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return xerrors.Errorf("commit tx: %w", err)
	}

	if err = transport.ServeJSONResponse(w, http.StatusOK, team); err != nil {
		return err
	}
	return nil
}

// HandleGet handles GET /teams/{id} request
//
// @Summary	Get team by id
// @Tags	Teams
// @Param	team_id	path		string	true	"Team id"
// @Success	200		{object}	storage.Team
// @Failure	400
// @Failure	404
// @Router	/teams/{team_id} [get]
func (h *Handler) HandleGet(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	teamID, err := transport.UUIDParam(r, "id")
	if err != nil {
		return xerrors.Errorf("%w", err)
	}

	s, err := h.factory.NewStorage(ctx, dbnode.Alive)
	if err != nil {
		return xerrors.Errorf("get storage: %w", err)
	}
	teamService := teams.NewService(s, h.inviteLinkPrefix)
	team, err := teamService.GetTeam(ctx, teamID)
	if err != nil {
		return xerrors.Errorf("get team %q: %w", teamID, err)
	}

	if err = transport.ServeJSONResponse(w, http.StatusOK, team); err != nil {
		return err
	}
	return nil
}

// HandleGetMany handles GET /quest/{id}/teams request
//
// @Summary	Get all teams by quest id
// @Tags	Teams
// @Param	quest_id	path		string	true	"Quest id"
// @Success	200			{object}	[]storage.Team
// @Failure	400
// @Router	/quest/{quest_id}/teams [get]
func (h *Handler) HandleGetMany(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	questID, err := transport.UUIDParam(r, "id")
	if err != nil {
		return xerrors.Errorf("%w", err)
	}

	s, err := h.factory.NewStorage(ctx, dbnode.Alive)
	if err != nil {
		return xerrors.Errorf("get storage: %w", err)
	}
	teamService := teams.NewService(s, h.inviteLinkPrefix)
	questTeams, err := teamService.GetQuestTeams(ctx, questID)
	if err != nil {
		return xerrors.Errorf("get teams of quest %q: %w", questID, err)
	}

	if err = transport.ServeJSONResponse(w, http.StatusOK, questTeams); err != nil {
		return err
	}
	return nil
}

type UpdateRequest struct {
	Name string `json:"name"`
}

// HandleUpdate handles POST /teams/{id} request
//
// @Summary		Change team information
// @Tags		Teams
// @Param		team_id	path		string			true	"Team id"
// @Param		request	body		UpdateRequest	true	"New information"
// @Success		200		{object} 	storage.Team
// @Failure    	400
// @Failure    	403
// @Failure		404
// @Router		/teams/{team_id} [post]
// @Security 	ApiKeyAuth
func (h *Handler) HandleUpdate(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	teamID, err := transport.UUIDParam(r, "id")
	if err != nil {
		return xerrors.Errorf("%w", err)
	}
	req, err := transport.UnmarshalRequestData[UpdateRequest](r)
	if err != nil {
		return xerrors.Errorf("%w", err)
	}
	uauth, err := jwt.GetUserFromContext(ctx)
	if err != nil {
		return xerrors.Errorf("%w", err)
	}

	s, err := h.factory.NewStorage(ctx, dbnode.Master)
	if err != nil {
		return xerrors.Errorf("get storage: %w", err)
	}
	teamService := teams.NewService(s, h.inviteLinkPrefix)
	team, err := teamService.UpdateTeamName(ctx, uauth, &storage.ChangeTeamNameRequest{ID: teamID, Name: req.Name})
	if err != nil {
		return xerrors.Errorf("update name: %w", err)
	}

	if err = transport.ServeJSONResponse(w, http.StatusOK, team); err != nil {
		return err
	}
	return nil
}

// HandleDelete handles DELETE /teams/{id} request
//
// @Summary		Delete team by id
// @Tags		Teams
// @Param		team_id	path	string	true	"Team id"
// @Success		200
// @Failure    	400
// @Failure    	403
// @Failure    	404
// @Router		/teams/{team_id} [delete]
// @Security 	ApiKeyAuth
func (h *Handler) HandleDelete(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	teamID, err := transport.UUIDParam(r, "id")
	if err != nil {
		return xerrors.Errorf("%w", err)
	}
	uauth, err := jwt.GetUserFromContext(ctx)
	if err != nil {
		return xerrors.Errorf("%w", err)
	}

	s, err := h.factory.NewStorage(ctx, dbnode.Master)
	if err != nil {
		return xerrors.Errorf("get storage: %w", err)
	}
	teamService := teams.NewService(s, h.inviteLinkPrefix)
	if err := teamService.DeleteTeam(ctx, uauth, &storage.DeleteTeamRequest{ID: teamID}); err != nil {
		return xerrors.Errorf("delete team %q: %w", teamID, err)
	}

	w.WriteHeader(http.StatusOK)
	return nil
}

type ChangeLeaderRequest struct {
	NewCaptainID string `json:"new_captain_id"`
}

// HandleChangeLeader handles POST /teams/{id}/captain request
//
// @Summary		Change team captain
// @Tags		Teams
// @Param		team_id	path		string				true	"Team id"
// @Param		request	body		ChangeLeaderRequest	true	"Change captain request"
// @Success		200		{object} 	storage.Team
// @Failure    	400
// @Failure    	403
// @Failure    	404
// @Router		/teams/{team_id}/captain [post]
// @Security 	ApiKeyAuth
func (h *Handler) HandleChangeLeader(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	teamID, err := transport.UUIDParam(r, "id")
	if err != nil {
		return xerrors.Errorf("%w", err)
	}
	req, err := transport.UnmarshalRequestData[ChangeLeaderRequest](r)
	if err != nil {
		return xerrors.Errorf("%w", err)
	}
	uauth, err := jwt.GetUserFromContext(ctx)
	if err != nil {
		return xerrors.Errorf("%w", err)
	}

	s, err := h.factory.NewStorage(ctx, dbnode.Master)
	if err != nil {
		return xerrors.Errorf("get storage: %w", err)
	}
	teamService := teams.NewService(s, h.inviteLinkPrefix)
	team, err := teamService.ChangeLeader(ctx, uauth, &storage.ChangeLeaderRequest{ID: teamID, CaptainID: req.NewCaptainID})
	if err != nil {
		return xerrors.Errorf("change captain of team %q to %q: %w", teamID, req.NewCaptainID, err)
	}

	if err = transport.ServeJSONResponse(w, http.StatusOK, team); err != nil {
		return err
	}
	return nil
}

// HandleLeave handles POST /teams/{id}/leave request
//
// @Summary		Leave the team
// @Tags		Teams
// @Param		team_id			path		string	true	"Team id"
// @Param		new_captain		query		string	false	"New captain (if leader leaves)"
// @Success		200				{object} 	storage.Team
// @Failure    	400
// @Failure    	403
// @Failure    	404
// @Router		/teams/{team_id}/leave [post]
// @Security 	ApiKeyAuth
func (h *Handler) HandleLeave(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	teamID, err := transport.UUIDParam(r, "id")
	if err != nil {
		return xerrors.Errorf("%w", err)
	}
	newCaptainID := transport.Query(r, "new_captain")
	uauth, err := jwt.GetUserFromContext(ctx)
	if err != nil {
		return xerrors.Errorf("%w", err)
	}

	s, tx, err := h.factory.NewStorageTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return xerrors.Errorf("start tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	teamService := teams.NewService(s, h.inviteLinkPrefix)
	team, err := teamService.LeaveTeam(ctx, uauth, teamID, newCaptainID)
	if err != nil {
		return xerrors.Errorf("leave team: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return xerrors.Errorf("commit tx: %w", err)
	}

	if err = transport.ServeJSONResponse(w, http.StatusOK, team); err != nil {
		return err
	}
	return nil
}

// HandleRemoveUser handles DELETE /teams/{id}/{user_id} request
//
// @Summary		Remove member from team
// @Tags		Teams
// @Param		team_id		path		string	true	"Team id"
// @Param		member_id	path		string	true	"Member id"
// @Success		200			{object} 	storage.Team
// @Failure    	400
// @Failure    	403
// @Failure    	404
// @Router		/teams/{team_id}/{member_id} [delete]
// @Security 	ApiKeyAuth
func (h *Handler) HandleRemoveUser(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	teamID, err := transport.UUIDParam(r, "id")
	if err != nil {
		return xerrors.Errorf("%w", err)
	}
	userID, err := transport.UUIDParam(r, "user_id")
	if err != nil {
		return xerrors.Errorf("%w", err)
	}
	uauth, err := jwt.GetUserFromContext(ctx)
	if err != nil {
		return xerrors.Errorf("%w", err)
	}

	s, err := h.factory.NewStorage(ctx, dbnode.Master)
	if err != nil {
		return xerrors.Errorf("get storage: %w", err)
	}
	teamService := teams.NewService(s, h.inviteLinkPrefix)
	team, err := teamService.RemoveUser(ctx, uauth, &storage.RemoveUserRequest{UserID: userID, ID: teamID})
	if err != nil {
		return xerrors.Errorf("remove user %q from team %q: %w", userID, teamID, err)
	}

	if err = transport.ServeJSONResponse(w, http.StatusOK, team); err != nil {
		return err
	}
	return nil
}

// HandleGetQuestByTeamInvite handles GET /join/{path}/quest request
//
// @Summary		Get quest by its team invite path
// @Tags		Teams
// @Param		invite_path	path		string	true	"Team invite path"
// @Success		200			{object}	quest.TeamQuestResponse
// @Failure		404
// @Router		/teams/join/{invite_path}/quest [get]
// @Security 	ApiKeyAuth
func (h *Handler) HandleGetQuestByTeamInvite(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	invitePath, _ := transport.StringParam(r, "path")
	s, err := h.factory.NewStorage(ctx, dbnode.Alive)
	if err != nil {
		return xerrors.Errorf("get storage client: %w", err)
	}
	team, err := s.GetTeam(ctx, &storage.GetTeamRequest{InvitePath: invitePath})
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return xerrors.Errorf("not found team with invite path %q: %w", invitePath, err)
		}
		return xerrors.Errorf("get team: %w", err)
	}

	req := storage.GetQuestRequest{ID: team.Quest.ID}
	gotQuest, err := s.GetQuest(ctx, &req)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return httperrors.Errorf(http.StatusNotFound, "not found quest with id %q", req.ID)
		}
		return xerrors.Errorf("get quest: %w", err)
	}

	quests.SetStatus(gotQuest)
	resp := quest.TeamQuestResponse{Quest: gotQuest}
	if uauth, err := jwt.GetUserFromContext(ctx); err == nil {
		teamReq := storage.GetTeamRequest{
			UserRegistration: &storage.UserRegistration{
				UserID:  uauth.ID,
				QuestID: team.Quest.ID,
			},
			IncludeMembers: true,
		}
		team, err := s.GetTeam(ctx, &teamReq)
		if err != nil && !errors.Is(err, storage.ErrNotFound) {
			return xerrors.Errorf("get user team: %w", err)
		}
		if team != nil {
			team.InviteLink = h.inviteLinkPrefix + team.InviteLink
		}
		resp.Team = team
	}

	if err = transport.ServeJSONResponse(w, http.StatusOK, &resp); err != nil {
		return err
	}
	return nil
}
