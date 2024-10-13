package play

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"golang.org/x/xerrors"

	"questspace/internal/pgdb"
	"questspace/internal/questspace/game"
	"questspace/internal/questspace/quests"
	"questspace/pkg/auth/jwt"
	"questspace/pkg/dbnode"
	"questspace/pkg/httperrors"
	"questspace/pkg/storage"
	"questspace/pkg/transport"
)

type Handler struct {
	clientFactory pgdb.QuestspaceClientFactory
}

func NewHandler(clientFactory pgdb.QuestspaceClientFactory) *Handler {
	return &Handler{
		clientFactory: clientFactory,
	}
}

type GetResponse struct {
	Quest      *storage.Quest      `json:"quest"`
	TaskGroups []storage.TaskGroup `json:"task_groups"`
	Team       *storage.Team       `json:"team"`
}

// HandleGet handles GET quest/:id/play request
//
// @Summary		Get task groups with tasks for play-mode
// @Tags		PlayMode
// @Param		quest_id	path		string		true	"Quest ID"
// @Success		200			{object}	game.AnswerDataResponse
// @Failure		400
// @Failure		401
// @Failure 	404
// @Failure 	406
// @Router		/quest/{id}/play [get]
// @Security 	ApiKeyAuth
func (h *Handler) HandleGet(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	questID, err := transport.UUIDParam(r, "id")
	if err != nil {
		return xerrors.Errorf("%w", err)
	}
	uauth, err := jwt.GetUserFromContext(ctx)
	if err != nil {
		return xerrors.Errorf("%w", err)
	}

	s, err := h.clientFactory.NewStorage(ctx, dbnode.Alive)
	if err != nil {
		return xerrors.Errorf("get storage: %w", err)
	}

	quest, err := s.GetQuest(ctx, &storage.GetQuestRequest{ID: questID})
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return httperrors.Errorf(http.StatusNotFound, "not found quest %q", questID)
		}
		return xerrors.Errorf("get quest: %w", err)
	}

	quests.SetStatus(quest)
	if quest.Status == storage.StatusOnRegistration || quest.Status == storage.StatusRegistrationDone {
		team, err := s.GetTeam(ctx, &storage.GetTeamRequest{
			UserRegistration: &storage.UserRegistration{
				UserID:  uauth.ID,
				QuestID: questID,
			},
		})
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				return httperrors.New(http.StatusForbidden, "cannot see brief message without registration")
			}
			return xerrors.Errorf("get team: %w", err)
		}
		if team.RegistrationStatus != storage.RegistrationStatusAccepted {
			return httperrors.New(http.StatusForbidden, "only accepted teams can view game mode")
		}
		resp := game.AnswerDataResponse{
			Quest: quest,
			Team:  team,
		}
		if err = transport.ServeJSONResponse(w, http.StatusOK, resp); err != nil {
			return err
		}
		return nil
	}

	if quest.Status != storage.StatusRunning {
		return httperrors.New(http.StatusNotAcceptable, "cannot get tasks before quest start")
	}
	taskGroups, err := s.GetTaskGroups(ctx, &storage.GetTaskGroupsRequest{QuestID: questID, IncludeTasks: true})
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return httperrors.Errorf(http.StatusNotFound, "not found quest %q", questID)
		}
		return xerrors.Errorf("get taskgroups: %w", err)
	}
	userTeam, err := s.GetTeam(ctx, &storage.GetTeamRequest{UserRegistration: &storage.UserRegistration{UserID: uauth.ID, QuestID: questID}})
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return httperrors.Errorf(http.StatusNotAcceptable, "user %q has no team", uauth.ID)
		}
		return xerrors.Errorf("get team: %w", err)
	}

	service := game.NewService(s, s, s, s)
	req := game.AnswerDataRequest{Quest: quest, Team: userTeam, TaskGroups: taskGroups}
	resp, err := service.FillAnswerData(ctx, &req)
	if err != nil {
		return xerrors.Errorf("fill answer data: %w", err)
	}

	if err = transport.ServeJSONResponse(w, http.StatusOK, resp); err != nil {
		return err
	}
	return nil
}

type TakeHintRequest struct {
	TaskID storage.ID `json:"task_id"`
	Index  int        `json:"index"`
}

// HandleTakeHint handles POST quest/:id/hint request
//
// @Summary		Take hint for task in play-mode
// @Tags		PlayMode
// @Param		quest_id	path		string					true	"Quest ID"
// @Param		request		body		play.TakeHintRequest	true	"Take hint request"
// @Success		200			{object}	storage.Hint
// @Failure		400
// @Failure		401
// @Failure 	404
// @Failure 	406
// @Router		/quest/{id}/hint [post]
// @Security 	ApiKeyAuth
func (h *Handler) HandleTakeHint(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	questID, err := transport.UUIDParam(r, "id")
	if err != nil {
		return xerrors.Errorf("%w", err)
	}
	req, err := transport.UnmarshalRequestData[TakeHintRequest](r)
	if err != nil {
		return xerrors.Errorf("%w", err)
	}
	uauth, err := jwt.GetUserFromContext(ctx)
	if err != nil {
		return xerrors.Errorf("%w", err)
	}

	s, tx, err := h.clientFactory.NewStorageTx(ctx, nil)
	if err != nil {
		return xerrors.Errorf("start tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	quest, err := s.GetQuest(ctx, &storage.GetQuestRequest{ID: questID})
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return httperrors.Errorf(http.StatusNotFound, "quest %q not found", questID)
		}
		return xerrors.Errorf("get quest: %w", err)
	}
	quests.SetStatus(quest)
	if quest.Status != storage.StatusRunning {
		return httperrors.New(http.StatusNotAcceptable, "cannot take hints before quest start")
	}

	srv := game.NewService(s, s, s, s)
	srvReq := game.TakeHintRequest{QuestID: questID, TaskID: req.TaskID, Index: req.Index}
	hint, err := srv.TakeHint(ctx, uauth, &srvReq)
	if err != nil {
		return xerrors.Errorf("hint error: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return xerrors.Errorf("commit tx: %w", err)
	}

	if err = transport.ServeJSONResponse(w, http.StatusOK, hint); err != nil {
		return err
	}
	return nil
}

type TryAnswerRequest struct {
	TaskID storage.ID
	Text   string
}

// HandleTryAnswer handles POST quest/:id/answer request
//
// @Summary		Answer task in play-mode
// @Tags		PlayMode
// @Param		quest_id	path		string					true	"Quest ID"
// @Param		request		body		play.TryAnswerRequest	true	"Answer data"
// @Success		200			{object}	game.TryAnswerResponse
// @Failure		400
// @Failure		401
// @Failure 	404
// @Failure 	406
// @Router		/quest/{id}/answer [post]
// @Security 	ApiKeyAuth
func (h *Handler) HandleTryAnswer(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	questID, err := transport.UUIDParam(r, "id")
	if err != nil {
		return xerrors.Errorf("%w", err)
	}
	req, err := transport.UnmarshalRequestData[TryAnswerRequest](r)
	if err != nil {
		return xerrors.Errorf("%w", err)
	}
	uauth, err := jwt.GetUserFromContext(ctx)
	if err != nil {
		return xerrors.Errorf("%w", err)
	}

	s, tx, err := h.clientFactory.NewStorageTx(ctx, nil)
	if err != nil {
		return xerrors.Errorf("start tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	quest, err := s.GetQuest(ctx, &storage.GetQuestRequest{ID: questID})
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return httperrors.Errorf(http.StatusNotFound, "quest %q not found", questID)
		}
		return xerrors.Errorf("get quest: %w", err)
	}
	quests.SetStatus(quest)
	if quest.Status != storage.StatusRunning {
		return httperrors.New(http.StatusNotAcceptable, "cannot take hints before quest start")
	}

	srv := game.NewService(s, s, s, s)
	srvReq := game.TryAnswerRequest{TaskID: req.TaskID, Text: req.Text, QuestID: questID}
	try, err := srv.TryAnswer(ctx, uauth, &srvReq)
	if err != nil {
		return xerrors.Errorf("try answer: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return xerrors.Errorf("commit tx: %w", err)
	}

	if err = transport.ServeJSONResponse(w, http.StatusOK, try); err != nil {
		return err
	}
	return nil
}

// HandleGetTableResults handles GET quest/:id/table request
//
// @Summary		Get admin leaderboard table during quest
// @Tags		PlayMode
// @Param		quest_id	path		string		true	"Quest ID"
// @Success		200			{object}	game.TeamResults
// @Failure		400
// @Failure		401
// @Failure 	403
// @Failure 	404
// @Router		/quest/{id}/table [get]
// @Security 	ApiKeyAuth
func (h *Handler) HandleGetTableResults(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	questID, err := transport.UUIDParam(r, "id")
	if err != nil {
		return xerrors.Errorf("%w", err)
	}
	uauth, err := jwt.GetUserFromContext(ctx)
	if err != nil {
		return xerrors.Errorf("%w", err)
	}

	s, err := h.clientFactory.NewStorage(ctx, dbnode.Alive)
	if err != nil {
		return xerrors.Errorf("get storage: %w", err)
	}
	quest, err := s.GetQuest(ctx, &storage.GetQuestRequest{ID: questID})
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return httperrors.Errorf(http.StatusNotFound, "quest %q not found", questID)
		}
		return xerrors.Errorf("get quest: %w", err)
	}
	if quest.Creator.ID != uauth.ID {
		return httperrors.New(http.StatusForbidden, "only creator can view leaderboard during quest time")
	}

	srv := game.NewService(s, s, s, s)
	leaderBoard, err := srv.GetResults(ctx, questID)
	if err != nil {
		return xerrors.Errorf("get results: %w", err)
	}

	if err = transport.ServeJSONResponse(w, http.StatusOK, leaderBoard); err != nil {
		return err
	}
	return nil
}

// HandleLeaderboard handles GET quest/:id/leaderboard request
//
// @Summary		Get leaderboard table with final results
// @Tags		PlayMode
// @Param		quest_id	path		string		true	"Quest ID"
// @Success		200			{object}	game.LeaderboardResponse
// @Failure		400
// @Failure		401
// @Failure 	403
// @Failure 	404
// @Router		/quest/{id}/leaderboard [get]
func (h *Handler) HandleLeaderboard(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	questID, err := transport.UUIDParam(r, "id")
	if err != nil {
		return xerrors.Errorf("%w", err)
	}

	s, err := h.clientFactory.NewStorage(ctx, dbnode.Alive)
	if err != nil {
		return xerrors.Errorf("get storage: %w", err)
	}
	quest, err := s.GetQuest(ctx, &storage.GetQuestRequest{ID: questID})
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return httperrors.Errorf(http.StatusNotFound, "quest %q not found", questID)
		}
		return xerrors.Errorf("get quest: %w", err)
	}
	quests.SetStatus(quest)
	if quest.Status != storage.StatusFinished {
		return httperrors.New(http.StatusNotFound, "leaderboard not ready yet")
	}

	srv := game.NewService(s, s, s, s)
	leaderBoard, err := srv.GetLeaderboard(ctx, questID)
	if err != nil {
		return xerrors.Errorf("get leaderboard: %w", err)
	}

	if err = transport.ServeJSONResponse(w, http.StatusOK, leaderBoard); err != nil {
		return err
	}
	return nil
}

// HandleAddPenalty handles POST quest/:id/penalty request
//
// @Summary		Add penalty to team
// @Tags		PlayMode
// @Param		quest_id	path		string					true	"Quest ID"
// @Param		request		body		game.AddPenaltyRequest	true	"Data to set penalty"
// @Success		200
// @Failure		400
// @Failure		401
// @Failure 	404
// @Failure 	406
// @Router		/quest/{id}/penalty [post]
// @Security 	ApiKeyAuth
func (h *Handler) HandleAddPenalty(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	questID, err := transport.UUIDParam(r, "id")
	if err != nil {
		return xerrors.Errorf("%w", err)
	}
	req, err := transport.UnmarshalRequestData[game.AddPenaltyRequest](r)
	if err != nil {
		return xerrors.Errorf("%w", err)
	}
	req.QuestID = questID
	uauth, err := jwt.GetUserFromContext(ctx)
	if err != nil {
		return xerrors.Errorf("%w", err)
	}

	s, tx, err := h.clientFactory.NewStorageTx(ctx, nil)
	if err != nil {
		return xerrors.Errorf("get storage: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	quest, err := s.GetQuest(ctx, &storage.GetQuestRequest{ID: questID})
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return httperrors.Errorf(http.StatusNotFound, "quest %q not found", questID)
		}
		return xerrors.Errorf("get quest: %w", err)
	}
	if quest.Creator.ID != uauth.ID {
		return httperrors.New(http.StatusForbidden, "only creator can add penalty to teams")
	}

	srv := game.NewService(s, s, s, s)
	if err = srv.AddPenalty(ctx, &req); err != nil {
		return xerrors.Errorf("add penalty: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return xerrors.Errorf("commit tx: %w", err)
	}
	w.WriteHeader(http.StatusOK)
	return nil
}

// HandleAnswerLog handles GET /quest/:id/answer_log request
//
// @Summary		Get paginated answer logs
// @Tags		PlayMode
// @Param      	task_group		query		string		false  "Task group ID"
// @Param		task			query		string		false  "Task ID"
// @Param		team			query		string		false  "Team ID"
// @Param      	user			query		string		false  "User ID"
// @Param      	accepted_only	query		bool		false  "Return only accepted answers"
// @Param		desc			query		bool        false  "Return new answers first (descending)"
// @Param		page_size		query		int			false  "Number of answers to return for each field"				default(50)
// @Param		page_no			query		int			false  "Page number to return. Mutually exclusive to page_id"
// @Param		page_id			query		string		false  "Page ID to return. Mutually exclusive to page_no"
// @Success		200				{object}	game.AnswerLogResponse
// @Failure		400
// @Failure		403
// @Failure		404
// @Router		/quest/{id}/answer_log [get]
// @Security	ApiKeyAuth
func (h *Handler) HandleAnswerLog(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	uauth, err := jwt.GetUserFromContext(ctx)
	if err != nil {
		return xerrors.Errorf("%w", err)
	}
	s, err := h.clientFactory.NewStorage(ctx, dbnode.Alive)
	if err != nil {
		return xerrors.Errorf("get storage: %w", err)
	}
	questID, err := transport.UUIDParam(r, "id")
	if err != nil {
		return xerrors.Errorf("%w", err)
	}
	quest, err := s.GetQuest(ctx, &storage.GetQuestRequest{ID: questID})
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return httperrors.Errorf(http.StatusNotFound, "quest %q not found", questID.String())
		}
		return xerrors.Errorf("get quest: %w", err)
	}
	if quest.Creator.ID != uauth.ID {
		return httperrors.New(http.StatusForbidden, "only creator can view answer log")
	}

	var opts []storage.FilteringOption
	if taskGroup := transport.Query(r, "task_group"); len(taskGroup) > 0 {
		opts = append(opts, storage.WithGroupID(storage.ID(taskGroup)))
	}
	if task := transport.Query(r, "task"); len(task) > 0 {
		opts = append(opts, storage.WithTaskID(storage.ID(task)))
	}
	if team := transport.Query(r, "team"); len(team) > 0 {
		opts = append(opts, storage.WithTeamID(storage.ID(team)))
	}
	if user := transport.Query(r, "user"); len(user) > 0 {
		opts = append(opts, storage.WithUserID(storage.ID(user)))
	}
	if accepted := transport.Query(r, "accepted_only"); len(accepted) > 0 {
		opts = append(opts, storage.WithOnlyAccepted())
	}
	if desc := transport.Query(r, "desc"); len(desc) > 0 {
		opts = append(opts, storage.WithDateDesc())
	}
	if pageSize := transport.Query(r, "page_size"); len(pageSize) > 0 {
		size, err := strconv.Atoi(pageSize)
		if err != nil {
			return httperrors.Errorf(http.StatusBadRequest, "bad page size: %w", err)
		}
		opts = append(opts, storage.WithPageSize(size))
	}
	if pageNo := transport.Query(r, "page_no"); len(pageNo) > 0 {
		num, err := strconv.Atoi(pageNo)
		if err != nil {
			return httperrors.Errorf(http.StatusBadRequest, "bad page_no: %w", err)
		}
		opts = append(opts, storage.WithPageNumber(num))
	}
	if pageID := transport.Query(r, "page_id"); len(pageID) > 0 {
		id, err := strconv.ParseInt(pageID, 10, 64)
		if err != nil {
			return httperrors.Errorf(http.StatusBadRequest, "bad page_id: %w", err)
		}
		opts = append(opts, storage.WithPageToken(id))
	}

	srv := game.NewService(s, s, s, s)
	logs, err := srv.GetAnswerLogs(ctx, uauth, questID, opts...)
	if err != nil {
		return xerrors.Errorf("get answer logs: %w", err)
	}

	if err = transport.ServeJSONResponse(w, http.StatusOK, &logs); err != nil {
		return err
	}
	return nil
}
