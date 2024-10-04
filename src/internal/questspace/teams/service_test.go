package teams

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/spkg/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"questspace/pkg/httperrors"
	"questspace/pkg/storage"
	storagemock "questspace/pkg/storage/mocks"
)

const linkPrefix = "link_starts_right__"

func TestService_CreateTeam(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)

	s := storagemock.NewMockTeamStorage(ctrl)
	service := NewService(s, linkPrefix)

	creator := storage.User{
		ID:        storage.NewID(),
		Username:  "svayp11",
		AvatarURL: "https://ya.ru",
	}
	questID := storage.NewID()

	req := storage.CreateTeamRequest{
		Creator: &creator,
		QuestID: questID,
		Name:    "new team",
	}

	createdTeam := storage.Team{
		ID:           storage.NewID(),
		Name:         req.Name,
		Captain:      &creator,
		Quest:        &storage.Quest{ID: questID, MaxTeamCap: ptr.Int(5)},
		InviteLinkID: 123,
	}
	inviteSuffix, err := LinkIDToPath(createdTeam.InviteLinkID)
	require.NoError(t, err)

	gomock.InOrder(
		s.EXPECT().
			GetTeams(ctx, &storage.GetTeamsRequest{User: &creator, QuestIDs: []storage.ID{questID}}).
			Return(nil, nil),

		s.EXPECT().CreateTeam(ctx, &req).Return(&createdTeam, nil),

		s.EXPECT().
			SetInviteLink(ctx, &storage.SetInvitePathRequest{TeamID: createdTeam.ID, InvitePath: inviteSuffix}).
			Return(nil),
	)

	team, err := service.CreateTeam(ctx, &req)
	require.NoError(t, err)
	assert.Truef(t, strings.HasPrefix(team.InviteLink, linkPrefix), "link does not start with prefix %q", linkPrefix)
	assert.Truef(t, strings.HasSuffix(team.InviteLink, inviteSuffix), "link does not end with invite path")
	require.Len(t, team.Members, 1)
	assert.Equal(t, creator, team.Members[0])
}

func TestTeamService_CreateTeam_AlreadyMember(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)

	s := storagemock.NewMockTeamStorage(ctrl)
	service := NewService(s, linkPrefix)

	creator := storage.User{
		ID:        storage.NewID(),
		Username:  "svayp11",
		AvatarURL: "https://ya.ru",
	}
	questID := storage.NewID()

	req := storage.CreateTeamRequest{
		Creator: &creator,
		QuestID: questID,
		Name:    "new team",
	}

	gomock.InOrder(
		s.EXPECT().
			GetTeams(ctx, &storage.GetTeamsRequest{User: &creator, QuestIDs: []storage.ID{questID}}).
			Return([]storage.Team{{}, {}, {}}, nil),
	)

	team, err := service.CreateTeam(ctx, &req)
	require.Error(t, err)
	assert.Nil(t, team)
	httpErr := new(httperrors.HTTPError)
	require.ErrorAs(t, err, &httpErr)
	assert.Equal(t, http.StatusNotAcceptable, httpErr.Code)
}

func TestTeamService_JoinTeam(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)

	s := storagemock.NewMockTeamStorage(ctrl)
	service := NewService(s, linkPrefix)

	newMember := storage.User{
		ID:        storage.NewID(),
		Username:  "svayp11",
		AvatarURL: "https://ya.ru",
	}
	req := storage.JoinTeamRequest{
		User:       &newMember,
		InvitePath: "inviteme",
	}

	questID := storage.NewID()
	teamCreator := storage.User{
		ID:       storage.NewID(),
		Username: "prikotletka",
	}
	team := storage.Team{
		ID:   storage.NewID(),
		Name: "team2",
		Quest: &storage.Quest{
			ID: questID,
		},
		InviteLink: req.InvitePath,
		Captain:    &teamCreator,
		Members:    []storage.User{teamCreator},
	}

	gomock.InOrder(
		s.EXPECT().
			GetTeam(ctx, &storage.GetTeamRequest{InvitePath: req.InvitePath, IncludeMembers: true}).
			Return(&team, nil),

		s.EXPECT().
			GetTeams(ctx, &storage.GetTeamsRequest{User: &newMember, QuestIDs: []storage.ID{questID}}).
			Return(nil, nil),

		s.EXPECT().JoinTeam(ctx, &req).Return(&newMember, nil),
	)

	got, err := service.JoinTeam(ctx, &req)
	require.NoError(t, err)
	assert.Equal(t, team.ID, got.ID)
	require.Len(t, got.Members, 2)
	assert.ElementsMatch(t, []storage.User{teamCreator, newMember}, got.Members)
}

func TestTeamService_JoinTeam_AlreadyInvited(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)

	s := storagemock.NewMockTeamStorage(ctrl)
	service := NewService(s, linkPrefix)

	oldMember := storage.User{
		ID:        storage.NewID(),
		Username:  "svayp11",
		AvatarURL: "https://ya.ru",
	}
	req := storage.JoinTeamRequest{
		User:       &oldMember,
		InvitePath: "inviteme",
	}

	questID := storage.NewID()
	teamCreator := storage.User{
		ID:       storage.NewID(),
		Username: "prikotletka",
	}
	team := storage.Team{
		ID:   storage.NewID(),
		Name: "team2",
		Quest: &storage.Quest{
			ID: questID,
		},
		InviteLink: req.InvitePath,
		Captain:    &teamCreator,
		Members:    []storage.User{teamCreator, oldMember},
	}

	gomock.InOrder(
		s.EXPECT().
			GetTeam(ctx, &storage.GetTeamRequest{InvitePath: req.InvitePath, IncludeMembers: true}).
			Return(&team, nil),
	)

	got, err := service.JoinTeam(ctx, &req)
	require.NoError(t, err)
	assert.Equal(t, team, *got)
}

func TestTeamService_LeaveTeam(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)

	s := storagemock.NewMockTeamStorage(ctrl)
	service := NewService(s, linkPrefix)

	oldMember := storage.User{
		ID:        storage.NewID(),
		Username:  "svayp11",
		AvatarURL: "https://ya.ru",
	}

	questID := storage.NewID()
	teamCreator := storage.User{
		ID:       storage.NewID(),
		Username: "prikotletka",
	}
	team := storage.Team{
		ID:   storage.NewID(),
		Name: "team2",
		Quest: &storage.Quest{
			ID: questID,
		},
		InviteLink: "inviteme",
		Captain:    &teamCreator,
		Members:    []storage.User{teamCreator, oldMember},
	}
	updatedTeam := team
	updatedTeam.Captain = &oldMember
	updatedTeam.Members = nil

	updatedTeamWithMembers := updatedTeam
	updatedTeamWithMembers.Members = []storage.User{oldMember}

	gomock.InOrder(
		s.EXPECT().GetTeam(ctx, &storage.GetTeamRequest{ID: team.ID, IncludeMembers: true}).
			Return(&team, nil),
		s.EXPECT().ChangeLeader(ctx, &storage.ChangeLeaderRequest{ID: team.ID, CaptainID: oldMember.ID}).
			Return(&updatedTeam, nil),
		s.EXPECT().RemoveUser(ctx, &storage.RemoveUserRequest{ID: team.ID, UserID: teamCreator.ID}).
			Return(nil),
		s.EXPECT().GetTeam(ctx, &storage.GetTeamRequest{ID: team.ID, IncludeMembers: true}).
			Return(&updatedTeamWithMembers, nil),
		s.EXPECT().RemoveUser(ctx, &storage.RemoveUserRequest{ID: team.ID, UserID: oldMember.ID}).
			Return(nil),
		s.EXPECT().DeleteTeam(ctx, &storage.DeleteTeamRequest{ID: team.ID}).Return(nil),
	)

	_, err := service.LeaveTeam(ctx, &teamCreator, team.ID, oldMember.ID)
	require.NoError(t, err)
	_, err = service.LeaveTeam(ctx, &oldMember, team.ID, "")
	require.NoError(t, err)
}
