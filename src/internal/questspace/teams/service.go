package teams

import (
	"context"
	"errors"
	"net/http"

	"go.uber.org/zap"

	"questspace/pkg/application/logging"

	"golang.org/x/xerrors"

	"questspace/pkg/application/httperrors"
	"questspace/pkg/storage"
)

type Service struct {
	s                storage.TeamStorage
	inviteLinkPrefix string
}

func NewService(s storage.TeamStorage, inviteLinkPrefix string) *Service {
	return &Service{
		s:                s,
		inviteLinkPrefix: inviteLinkPrefix,
	}
}

func (s *Service) CreateTeam(ctx context.Context, req *storage.CreateTeamRequest) (*storage.Team, error) {
	exisingTeams, err := s.s.GetTeams(ctx, &storage.GetTeamsRequest{User: req.Creator, QuestIDs: []string{req.QuestID}})
	if err != nil {
		return nil, xerrors.Errorf("get existing teams for user %s: %w", req.Creator.ID, err)
	}
	if len(exisingTeams) > 0 {
		return nil, httperrors.New(http.StatusNotAcceptable, "cannot create more than one team for quest")
	}
	team, err := s.s.CreateTeam(ctx, req)
	if err != nil {
		return nil, xerrors.Errorf("create team: %w", err)
	}
	invitePath, err := LinkIDToPath(team.InviteLinkID)
	if err != nil {
		return nil, xerrors.Errorf("create invite link: %w", err)
	}
	if err := s.s.SetInviteLink(ctx, &storage.SetInvitePathRequest{InvitePath: invitePath, TeamID: team.ID}); err != nil {
		return nil, xerrors.Errorf("save invite url: %w", err)
	}
	team.InviteLink = s.inviteLinkPrefix + invitePath
	team.Members = append(team.Members, *req.Creator)
	return team, nil
}

func (s *Service) GetTeam(ctx context.Context, teamID string) (*storage.Team, error) {
	team, err := s.s.GetTeam(ctx, &storage.GetTeamRequest{ID: teamID, IncludeMembers: true})
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, httperrors.New(http.StatusNotFound, "team not found")
		}
		return nil, xerrors.Errorf("get team: %w", err)
	}
	team.InviteLink = s.inviteLinkPrefix + team.InviteLink
	return team, nil
}

func (s *Service) GetQuestTeams(ctx context.Context, questID string) ([]storage.Team, error) {
	teams, err := s.s.GetTeams(ctx, &storage.GetTeamsRequest{QuestIDs: []string{questID}})
	if err != nil {
		return nil, xerrors.Errorf("get teams: %w", err)
	}
	for _, t := range teams {
		t.InviteLink = s.inviteLinkPrefix + t.InviteLink
	}
	return teams, nil
}

func (s *Service) UpdateTeamName(ctx context.Context, user *storage.User, req *storage.ChangeTeamNameRequest) (*storage.Team, error) {
	team, err := s.s.GetTeam(ctx, &storage.GetTeamRequest{ID: req.ID, IncludeMembers: true})
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, httperrors.New(http.StatusNotFound, "team not found")
		}
		return nil, xerrors.Errorf("get team information: %w", err)
	}

	if team.Captain.ID != user.ID {
		return nil, httperrors.New(http.StatusForbidden, "only captain can change team name")
	}

	newTeam, err := s.s.ChangeTeamName(ctx, req)
	if err != nil {
		return nil, xerrors.Errorf("change team name: %w", err)
	}
	newTeam.InviteLink = s.inviteLinkPrefix + newTeam.InviteLink
	return newTeam, nil
}

func (s *Service) JoinTeam(ctx context.Context, req *storage.JoinTeamRequest) (*storage.Team, error) {
	team, err := s.s.GetTeam(ctx, &storage.GetTeamRequest{InvitePath: req.InvitePath, IncludeMembers: true})
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, httperrors.New(http.StatusNotFound, "team not found")
		}
		return nil, xerrors.Errorf("get team information: %w", err)
	}
	for _, member := range team.Members {
		if member.ID == req.User.ID {
			return team, nil
		}
	}

	exisingTeams, err := s.s.GetTeams(ctx, &storage.GetTeamsRequest{User: req.User, QuestIDs: []string{team.Quest.ID}})
	if err != nil {
		return nil, xerrors.Errorf("get existing teams for user %s: %w", req.User.ID, err)
	}
	if len(exisingTeams) > 0 {
		return nil, httperrors.New(http.StatusNotAcceptable, "cannot join more than one team for quest")
	}

	if team.Quest.MaxTeamCap != nil && *team.Quest.MaxTeamCap == len(team.Members) {
		return nil, httperrors.New(http.StatusNotAcceptable, "already max amount of users were registered")
	}
	user, err := s.s.JoinTeam(ctx, req)
	if err != nil {
		if errors.Is(err, storage.ErrTeamAlreadyFull) {
			return nil, httperrors.New(http.StatusNotAcceptable, "team already full")
		}
		return nil, xerrors.Errorf("join team by link %q: %w", req.InvitePath, err)
	}
	team.InviteLink = s.inviteLinkPrefix + team.InviteLink
	team.Members = append(team.Members, *user)
	return team, nil
}

func (s *Service) DeleteTeam(ctx context.Context, user *storage.User, req *storage.DeleteTeamRequest) error {
	team, err := s.s.GetTeam(ctx, &storage.GetTeamRequest{ID: req.ID})
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return httperrors.New(http.StatusNotFound, "team not found")
		}
		return xerrors.Errorf("get team: %w", err)
	}
	if team.Captain.ID != user.ID {
		return httperrors.New(http.StatusForbidden, "only team captain can delete their team")
	}
	if err := s.s.DeleteTeam(ctx, req); err != nil {
		return xerrors.Errorf("delete team: %w", err)
	}
	return nil
}

func (s *Service) ChangeLeader(ctx context.Context, user *storage.User, req *storage.ChangeLeaderRequest) (*storage.Team, error) {
	team, err := s.s.GetTeam(ctx, &storage.GetTeamRequest{ID: req.ID, IncludeMembers: true})
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, httperrors.Errorf(http.StatusNotFound, "team %q not found", req.ID)
		}
		return nil, xerrors.Errorf("get team: %w", err)
	}
	if team.Captain.ID != user.ID {
		return nil, httperrors.New(http.StatusForbidden, "only captain can pass their role to another member")
	}

	newTeam, err := s.s.ChangeLeader(ctx, req)
	if err != nil {
		return nil, xerrors.Errorf("change leader: %w", err)
	}
	newTeam.InviteLink = s.inviteLinkPrefix + newTeam.InviteLink

	return newTeam, nil
}

func (s *Service) LeaveTeam(ctx context.Context, user *storage.User, teamID, newCaptainID string) (*storage.Team, error) {
	team, err := s.s.GetTeam(ctx, &storage.GetTeamRequest{ID: teamID, IncludeMembers: true})
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, httperrors.New(http.StatusNotFound, "team not found")
		}
		return nil, xerrors.Errorf("get team: %w", err)
	}
	newTeam := team

	if team.Captain.ID == user.ID {
		if newCaptainID == "" {
			return nil, httperrors.New(http.StatusBadRequest, "captain cannot leave team without specifying next leader")
		}
		changeCapReq := storage.ChangeLeaderRequest{
			ID:        teamID,
			CaptainID: newCaptainID,
		}
		var err error
		newTeam, err = s.s.ChangeLeader(ctx, &changeCapReq)
		if err != nil {
			return nil, xerrors.Errorf("change captain: %w", err)
		}

		logging.Info(ctx, "new captain",
			zap.String("team_id", teamID),
			zap.String("old_captain", user.ID),
			zap.String("new_captain", newCaptainID),
		)
	}

	if err := s.s.RemoveUser(ctx, &storage.RemoveUserRequest{ID: teamID, UserID: user.ID}); err != nil {
		return nil, xerrors.Errorf("leave team: %w", err)
	}

	members := make([]storage.User, 0, len(team.Members)-1)
	for _, member := range team.Members {
		if member.ID == user.ID {
			continue
		}
		members = append(members, member)
	}
	newTeam.Members = members
	newTeam.InviteLink = s.inviteLinkPrefix + newTeam.InviteLink
	return newTeam, nil
}

func (s *Service) RemoveUser(ctx context.Context, user *storage.User, req *storage.RemoveUserRequest) (*storage.Team, error) {
	team, err := s.s.GetTeam(ctx, &storage.GetTeamRequest{ID: req.ID, IncludeMembers: true})
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, httperrors.New(http.StatusNotFound, "team not found")
		}
		return nil, xerrors.Errorf("get team: %w", err)
	}
	if team.Captain.ID != user.ID {
		return nil, httperrors.New(http.StatusForbidden, "only team captain can remove members")
	}
	if req.UserID == user.ID {
		return nil, httperrors.New(http.StatusBadRequest, "cannot leave team without specifying new captain")
	}

	if err := s.s.RemoveUser(ctx, req); err != nil {
		return nil, xerrors.Errorf("remove user: %w", err)
	}

	members := make([]storage.User, 0, len(team.Members)-1)
	for _, member := range team.Members {
		if member.ID == req.UserID {
			continue
		}
		members = append(members, member)
	}
	team.Members = members
	team.InviteLink = s.inviteLinkPrefix + team.InviteLink

	return team, nil
}
