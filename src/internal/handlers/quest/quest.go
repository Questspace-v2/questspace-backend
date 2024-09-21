package quest

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"go.uber.org/zap"
	"golang.org/x/xerrors"

	"questspace/internal/pgdb"
	"questspace/internal/questspace/game"
	"questspace/internal/questspace/quests"
	"questspace/internal/validate"
	"questspace/pkg/auth/jwt"
	"questspace/pkg/dbnode"
	"questspace/pkg/httperrors"
	"questspace/pkg/logging"
	"questspace/pkg/storage"
	"questspace/pkg/transport"
)

type Handler struct {
	clientFactory    pgdb.QuestspaceClientFactory
	fetcher          http.Client
	inviteLinkPrefix string
}

func NewHandler(cf pgdb.QuestspaceClientFactory, f http.Client, p string) *Handler {
	return &Handler{
		clientFactory:    cf,
		fetcher:          f,
		inviteLinkPrefix: p,
	}
}

// HandleCreate handles POST /quest request
//
// @Summary		Create new quest
// @Tags		Quests
// @Param		request	body		storage.CreateQuestRequest	true	"Main quest information"
// @Success		200		{object}	storage.Quest
// @Failure		400
// @Failure		401
// @Failure		415
// @Router		/quest [post]
// @Security	ApiKeyAuth
func (h *Handler) HandleCreate(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	req, err := transport.UnmarshalRequestData[storage.CreateQuestRequest](r)
	if err != nil {
		return xerrors.Errorf("%w", err)
	}
	uauth, err := jwt.GetUserFromContext(ctx)
	if err != nil {
		return xerrors.Errorf("%w", err)
	}
	req.Creator = uauth
	if err := validate.ImageURL(ctx, h.fetcher, req.MediaLink); err != nil {
		return xerrors.Errorf("%w", err)
	}

	s, err := h.clientFactory.NewStorage(ctx, dbnode.Master)
	if err != nil {
		return xerrors.Errorf("get storage client: %w", err)
	}
	quest, err := s.CreateQuest(ctx, req)
	if err != nil {
		return xerrors.Errorf("create quest: %w", err)
	}
	quests.SetStatus(quest)
	if err = transport.ServeJSONResponse(w, http.StatusOK, quest); err != nil {
		return err
	}

	logging.Info(ctx, "created quest",
		zap.String("quest_id", quest.ID),
		zap.String("quest_name", quest.Name),
		zap.String("creator_id", uauth.ID),
		zap.String("creator_name", uauth.Username),
	)
	return nil
}

type TeamQuestResponse struct {
	Quest       *storage.Quest            `json:"quest"`
	Team        *storage.Team             `json:"team,omitempty"`
	Leaderboard *game.LeaderboardResponse `json:"leaderboard,omitempty"`
}

// HandleGet handles GET /quest/:id request
//
// @Summary		Get quest by id
// @Tags		Quests
// @Param		quest_id	path		string	true	"Quest ID"
// @Success		200			{object}	quest.TeamQuestResponse
// @Failure		404
// @Router		/quest/{quest_id} [get]
// @Security 	ApiKeyAuth
func (h *Handler) HandleGet(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	questID, err := transport.UUIDParam(r, "id")
	if err != nil {
		return xerrors.Errorf("%w", err)
	}
	req := storage.GetQuestRequest{ID: questID}
	s, err := h.clientFactory.NewStorage(ctx, dbnode.Alive)
	if err != nil {
		return xerrors.Errorf("get storage client: %w", err)
	}
	quest, err := s.GetQuest(ctx, &req)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return httperrors.Errorf(http.StatusNotFound, "not found quest with id %q", req.ID)
		}
		return xerrors.Errorf("get quest: %w", err)
	}
	hasBrief := quest.HasBrief
	quest.HasBrief = false
	quest.Brief = ""

	quests.SetStatus(quest)
	resp := TeamQuestResponse{Quest: quest}
	if uauth, err := jwt.GetUserFromContext(ctx); err == nil {
		teamReq := storage.GetTeamRequest{
			UserRegistration: &storage.UserRegistration{
				UserID:  uauth.ID,
				QuestID: questID,
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

	if quest.Status == storage.StatusFinished {
		srv := game.NewService(s, s, s, s)
		leaderboard, err := srv.GetLeaderboard(ctx, questID)
		if err == nil {
			resp.Leaderboard = leaderboard
		} else {
			logging.Error(ctx, "get leaderboard", zap.Error(err))
		}
	}
	if (quest.Status == storage.StatusOnRegistration || quest.Status == storage.StatusRegistrationDone) && resp.Team != nil {
		resp.Quest.HasBrief = hasBrief
	}

	if err = transport.ServeJSONResponse(w, http.StatusOK, resp); err != nil {
		return err
	}
	return nil
}

const defaultPageSize = 50

// HandleGetMany handles GET /quest request
//
// @Summary		Get many quests sorted by start time and finished status
// @Tags		Quests
// @Param      	fields		query		[]string   false  "Fields to return"  Enums(all, registered, owned) minlength(0) maxlength(3)
// @Param		page_size	query		int        false  "Number of quests to return for each field" default(50)
// @Param		page_id		query		string     false  "Page ID to return. Mutually exclusive to multiple fields"
// @Success		200			{object}	quests.Quests
// @Failure		400
// @Router		/quest [get]
// @Security	ApiKeyAuth
func (h *Handler) HandleGetMany(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	uauth, _ := jwt.GetUserFromContext(ctx)
	fields := transport.QueryArray(r, "fields")
	if uauth == nil {
		fields = []string{"all"}
	}

	pageSizeStr := transport.Query(r, "page_size")
	pageSize := defaultPageSize
	if pageSizeStr != "" {
		var err error
		pageSize, err = strconv.Atoi(pageSizeStr)
		if err != nil {
			return httperrors.Errorf(http.StatusBadRequest, "parse page size: %w", err)
		}
	}
	pageID := transport.Query(r, "page_id")

	s, err := h.clientFactory.NewStorage(ctx, dbnode.Alive)
	if err != nil {
		return xerrors.Errorf("get storage: %w", err)
	}
	gotQuests, err := quests.ReadQuests(ctx, s, uauth, fields, pageID, pageSize)
	if err != nil {
		return xerrors.Errorf("read quests: %w", err)
	}

	if err = transport.ServeJSONResponse(w, http.StatusOK, gotQuests); err != nil {
		return err
	}
	return nil
}

// HandleUpdate handles POST /quest/:id request
//
// @Summary		Update main quest information
// @Tags 		Quests
// @Param		quest_id	path		string						true	"Quest ID"
// @Param		request		body		storage.UpdateQuestRequest	true	"Quest information to update"
// @Success		200			{object}	storage.Quest
// @Failure    	401
// @Failure    	403
// @Failure		404
// @Failure		415
// @Router		/quest/{quest_id} [post]
// @Security	ApiKeyAuth
func (h *Handler) HandleUpdate(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	req, err := transport.UnmarshalRequestData[storage.UpdateQuestRequest](r)
	if err != nil {
		return xerrors.Errorf("%w", err)
	}
	req.ID, err = transport.UUIDParam(r, "id")
	if err != nil {
		return xerrors.Errorf("%w", err)
	}
	if err := validate.ImageURL(ctx, h.fetcher, req.MediaLink); err != nil {
		return xerrors.Errorf("%w", err)
	}
	uauth, err := jwt.GetUserFromContext(ctx)
	if err != nil {
		return xerrors.Errorf("%w", err)
	}

	s, tx, err := h.clientFactory.NewStorageTx(ctx, nil)
	if err != nil {
		return xerrors.Errorf("get storage: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	quest, err := s.UpdateQuest(ctx, req)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return httperrors.Errorf(http.StatusNotFound, "not found quest with id %q", req.ID)
		}
		return xerrors.Errorf("failed to update quest: %w", err)
	}
	if quest.Creator == nil || quest.Creator.ID != uauth.ID {
		return httperrors.New(http.StatusForbidden, "only creator can update their quest")
	}
	if err := tx.Commit(); err != nil {
		return xerrors.Errorf("commit transaction: %w", err)
	}
	quests.SetStatus(quest)
	if err = transport.ServeJSONResponse(w, http.StatusOK, quest); err != nil {
		return err
	}
	return nil
}

// HandleDelete handles DELETE /quest/:id request
//
// @Summary		Delete quest
// @Tags 		Quests
// @Param		quest_id	path	string	true	"Quest ID"
// @Success		200
// @Failure    	401
// @Failure    	403
// @Failure    	404
// @Router		/quest/{quest_id} [delete]
// @Security 	ApiKeyAuth
func (h *Handler) HandleDelete(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	id, err := transport.UUIDParam(r, "id")
	if err != nil {
		return xerrors.Errorf("%w", err)
	}
	uauth, err := jwt.GetUserFromContext(ctx)
	if err != nil {
		return xerrors.Errorf("%w", err)
	}
	s, err := h.clientFactory.NewStorage(ctx, dbnode.Master)
	if err != nil {
		return xerrors.Errorf("get storage client: %w", err)
	}

	q, err := s.GetQuest(ctx, &storage.GetQuestRequest{ID: id})
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return httperrors.Errorf(http.StatusNotFound, "quest %q not found", id)
		}
		return xerrors.Errorf("get quest: %w", err)
	}
	if q.Creator == nil || q.Creator.ID != uauth.ID {
		return httperrors.New(http.StatusForbidden, "cannot delete others' quests")
	}

	if err = s.DeleteQuest(ctx, &storage.DeleteQuestRequest{ID: id}); err != nil {
		return xerrors.Errorf("%w", err)
	}
	w.WriteHeader(http.StatusOK)
	return nil
}

// HandleFinish handles POST /quest/:id/finish request
//
// @Summary		Finish quest
// @Tags 		Quests
// @Param		quest_id	path	string	true	"Quest ID"
// @Success		200
// @Failure    	401
// @Failure    	403
// @Failure    	404
// @Router		/quest/{quest_id}/finish [post]
// @Security 	ApiKeyAuth
func (h *Handler) HandleFinish(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	id, err := transport.UUIDParam(r, "id")
	if err != nil {
		return xerrors.Errorf("%w", err)
	}
	uauth, err := jwt.GetUserFromContext(ctx)
	if err != nil {
		return xerrors.Errorf("%w", err)
	}
	s, err := h.clientFactory.NewStorage(ctx, dbnode.Master)
	if err != nil {
		return xerrors.Errorf("get storage client: %w", err)
	}

	q, err := s.GetQuest(ctx, &storage.GetQuestRequest{ID: id})
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return httperrors.Errorf(http.StatusNotFound, "quest %q not found", id)
		}
		return xerrors.Errorf("get quest: %w", err)
	}
	if q.Creator == nil || q.Creator.ID != uauth.ID {
		return httperrors.New(http.StatusForbidden, "only creator can finish their quests")
	}

	if err = s.FinishQuest(ctx, &storage.FinishQuestRequest{ID: id}); err != nil {
		return xerrors.Errorf("finish quest: %w", err)
	}
	w.WriteHeader(http.StatusOK)
	return nil
}
