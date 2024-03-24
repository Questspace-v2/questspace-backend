package teams

import (
	"context"
	"errors"
	"net/http"

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

func (t *Service) CreateTeam(ctx context.Context, req *storage.CreateTeamRequest) (*storage.Team, error) {
	exisingTeams, err := t.s.GetTeams(ctx, &storage.GetTeamsRequest{User: req.Creator, QuestIDs: []string{req.QuestID}})
	if err != nil {
		return nil, xerrors.Errorf("get existing teams for user %s: %w", req.Creator.ID, err)
	}
	if len(exisingTeams) > 0 {
		return nil, httperrors.New(http.StatusNotAcceptable, "cannot create more than one team for quest")
	}
	team, err := t.s.CreateTeam(ctx, req)
	if err != nil {
		return nil, xerrors.Errorf("create team: %w", err)
	}
	invitePath, err := LinkIDToPath(team.InviteLinkID)
	if err != nil {
		return nil, xerrors.Errorf("create invite link: %w", err)
	}
	if err := t.s.SetInviteLink(ctx, &storage.SetInviteLinkRequest{InviteURL: invitePath, TeamID: team.ID}); err != nil {
		return nil, xerrors.Errorf("save invite url: %w", err)
	}
	team.InviteLink = t.inviteLinkPrefix + invitePath
	team.Members = append(team.Members, req.Creator)
	return team, nil
}

func (t *Service) JoinTeam(ctx context.Context, req *storage.JoinTeamRequest) (*storage.Team, error) {
	team, err := t.s.GetTeam(ctx, &storage.GetTeamRequest{InvitePath: req.InvitePath, IncludeMembers: true})
	if err != nil {
		return nil, xerrors.Errorf("get team information: %w", err)
	}
	for _, member := range team.Members {
		if member.ID == req.User.ID {
			return team, nil
		}
	}

	exisingTeams, err := t.s.GetTeams(ctx, &storage.GetTeamsRequest{User: req.User, QuestIDs: []string{team.Quest.ID}})
	if err != nil {
		return nil, xerrors.Errorf("get existing teams for user %s: %w", req.User.ID, err)
	}
	if len(exisingTeams) > 0 {
		return nil, httperrors.New(http.StatusNotAcceptable, "cannot join more than one team for quest")
	}

	if team.Quest.MaxTeamCap != nil && *team.Quest.MaxTeamCap == len(team.Members) {
		return nil, httperrors.New(http.StatusNotAcceptable, "already max amount of users were registered")
	}
	user, err := t.s.JoinTeam(ctx, req)
	if err != nil {
		if errors.Is(err, storage.ErrTeamAlreadyFull) {
			return nil, httperrors.New(http.StatusNotAcceptable, "team already full")
		}
		return nil, xerrors.Errorf("join team by link %q: %w", req.InvitePath, err)
	}
	team.InviteLink = t.inviteLinkPrefix + team.InviteLink
	team.Members = append(team.Members, user)
	return team, nil
}
