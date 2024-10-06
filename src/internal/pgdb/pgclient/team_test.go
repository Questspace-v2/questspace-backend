package pgclient

import (
	"context"
	"testing"
	"time"

	"github.com/spkg/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"questspace/internal/pgdb/pgtest"
	"questspace/pkg/storage"
)

var (
	userReq1 = &storage.CreateUserRequest{
		Username:  "svayp11",
		Password:  "veryverysecure",
		AvatarURL: "https://google.com",
	}
	userReq2 = &storage.CreateUserRequest{
		Username:  "prikotletka",
		Password:  "evenmoresecure",
		AvatarURL: "https://ya.ru",
	}

	questReq1 = storage.CreateQuestRequest{
		Name:      "newquest",
		StartTime: ptr.Time(time.Now()),
		Access:    storage.AccessPublic,
	}
	questReq2 = storage.CreateQuestRequest{
		Name:      "newerquest",
		StartTime: ptr.Time(time.Now()),
		Access:    storage.AccessPublic,
	}

	teamReq1 = storage.CreateTeamRequest{
		Name: "my_great_team",
	}
	teamReq2 = storage.CreateTeamRequest{
		Name: "my_even_greater_team",
	}
	teamReq3 = storage.CreateTeamRequest{
		Name: "team",
	}

	firstPath  = "first"
	secondPath = "second"
)

func TestTeamStorage_CreateTeam_Success(t *testing.T) {
	ctx := context.Background()
	client := NewClient(pgtest.NewEmbeddedQuestspaceDB(t))
	user, err := client.CreateUser(ctx, userReq1)
	require.NoError(t, err)

	qReq := questReq1
	qReq.Creator = user
	q, err := client.CreateQuest(ctx, &qReq)
	require.NoError(t, err)

	teamReq := teamReq1
	teamReq.Creator = user
	teamReq.QuestID = q.ID
	team, err := client.CreateTeam(ctx, &teamReq)
	require.NoError(t, err)

	assert.Equal(t, teamReq.Name, team.Name)
	assert.Equal(t, teamReq.Creator.ID, team.Captain.ID)
	assert.Equal(t, teamReq.Creator.AvatarURL, team.Captain.AvatarURL)
	assert.Equal(t, teamReq.Creator.Username, team.Captain.Username)
	assert.Nil(t, team.Quest)

	members, err := client.getTeamMembers(ctx, team.ID)
	require.NoError(t, err)
	require.Len(t, members, 1)
	assert.Equal(t, team.Captain.ID, members[0].ID)
	assert.Equal(t, team.Captain.Username, members[0].Username)
	assert.Equal(t, team.Captain.AvatarURL, members[0].AvatarURL)
}

func TestTeamStorage_AlreadyExists(t *testing.T) {
	ctx := context.Background()
	client := NewClient(pgtest.NewEmbeddedQuestspaceDB(t))

	user1, err := client.CreateUser(ctx, userReq1)
	require.NoError(t, err)
	user2, err := client.CreateUser(ctx, userReq2)
	require.NoError(t, err)

	qReq := questReq1
	qReq.Creator = user1
	q, err := client.CreateQuest(ctx, &qReq)
	require.NoError(t, err)

	teamReq := teamReq1
	teamReq.Creator = user1
	teamReq.QuestID = q.ID
	team1, err := client.CreateTeam(ctx, &teamReq)
	require.NoError(t, err)
	assert.NotNil(t, team1)

	team2, err := client.CreateTeam(ctx, &storage.CreateTeamRequest{
		Name:    teamReq.Name,
		QuestID: teamReq.QuestID,
		Creator: user2,
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, storage.ErrExists)
	assert.Nil(t, team2)
}

func TestTeamStorage_GetTeams(t *testing.T) {
	ctx := context.Background()
	client := NewClient(pgtest.NewEmbeddedQuestspaceDB(t))

	user1, err := client.CreateUser(ctx, userReq1)
	require.NoError(t, err)
	user2, err := client.CreateUser(ctx, userReq2)
	require.NoError(t, err)

	qReq1 := questReq1
	qReq1.Creator = user1
	q1, err := client.CreateQuest(ctx, &qReq1)
	require.NoError(t, err)

	qReq2 := questReq2
	qReq2.Creator = user2
	q2, err := client.CreateQuest(ctx, &qReq2)
	require.NoError(t, err)

	tReq1 := teamReq1
	tReq1.Creator = user1
	tReq1.QuestID = q1.ID
	team1, err := client.CreateTeam(ctx, &tReq1)
	require.NoError(t, err)

	tReq2 := teamReq2
	tReq2.Creator = user2
	tReq2.QuestID = q1.ID
	team2, err := client.CreateTeam(ctx, &tReq2)
	require.NoError(t, err)

	tReq3 := teamReq3
	tReq3.Creator = user1
	tReq3.QuestID = q2.ID
	team3, err := client.CreateTeam(ctx, &tReq3)
	require.NoError(t, err)

	u1q1Teams, err := client.GetTeams(ctx, &storage.GetTeamsRequest{User: user1, QuestIDs: []storage.ID{q1.ID}})
	require.NoError(t, err)
	require.Len(t, u1q1Teams, 1)
	assert.Equal(t, team1.ID, u1q1Teams[0].ID)

	u1q2Teams, err := client.GetTeams(ctx, &storage.GetTeamsRequest{User: user1, QuestIDs: []storage.ID{q2.ID}})
	require.NoError(t, err)
	require.Len(t, u1q2Teams, 1)
	assert.Equal(t, team3.ID, u1q2Teams[0].ID)

	u2AllTeams, err := client.GetTeams(ctx, &storage.GetTeamsRequest{User: user2})
	require.NoError(t, err)
	require.Len(t, u2AllTeams, 1)
	assert.Equal(t, team2.ID, u2AllTeams[0].ID)

	u1AllTeams, err := client.GetTeams(ctx, &storage.GetTeamsRequest{User: user1})
	require.NoError(t, err)
	require.Len(t, u1AllTeams, 2)
}

func TestTeamStorage_GetTeam(t *testing.T) {
	ctx := context.Background()
	client := NewClient(pgtest.NewEmbeddedQuestspaceDB(t))

	user1, err := client.CreateUser(ctx, userReq1)
	require.NoError(t, err)
	user2, err := client.CreateUser(ctx, userReq2)
	require.NoError(t, err)

	qReq1 := questReq1
	qReq1.Creator = user1
	q1, err := client.CreateQuest(ctx, &qReq1)
	require.NoError(t, err)

	tReq1 := teamReq1
	tReq1.Creator = user1
	tReq1.QuestID = q1.ID
	team1, err := client.CreateTeam(ctx, &tReq1)
	require.NoError(t, err)
	require.NoError(t, client.SetInviteLink(ctx, &storage.SetInvitePathRequest{
		TeamID:     team1.ID,
		InvitePath: firstPath,
	}))
	team1.InviteLink = firstPath

	tReq2 := teamReq2
	tReq2.Creator = user2
	tReq2.QuestID = q1.ID
	team2, err := client.CreateTeam(ctx, &tReq2)
	require.NoError(t, err)
	require.NoError(t, client.SetInviteLink(ctx, &storage.SetInvitePathRequest{
		TeamID:     team2.ID,
		InvitePath: secondPath,
	}))
	team2.InviteLink = secondPath

	recvTeamByID1, err := client.GetTeam(ctx, &storage.GetTeamRequest{ID: team1.ID})
	require.NoError(t, err)
	require.NotNil(t, recvTeamByID1.Quest.Creator)
	assert.NotEmpty(t, recvTeamByID1.Quest.Creator.ID)

	recvTeamByURL1, err := client.GetTeam(ctx, &storage.GetTeamRequest{InvitePath: team1.InviteLink})
	require.NoError(t, err)
	assert.Equal(t, recvTeamByURL1, recvTeamByID1)
	assert.Nil(t, recvTeamByID1.Quest.MaxTeamCap)
	assert.Equal(t, team1.ID, recvTeamByID1.ID)
	assert.Equal(t, team1.Name, recvTeamByID1.Name)
	assert.Equal(t, team1.Captain.ID, recvTeamByID1.Captain.ID)
	assert.Equal(t, team1.Captain.Username, recvTeamByID1.Captain.Username)
	assert.Equal(t, team1.Captain.AvatarURL, recvTeamByID1.Captain.AvatarURL)
	assert.Equal(t, q1.Creator.ID, recvTeamByID1.Quest.Creator.ID)

	recvTeamByID2, err := client.GetTeam(ctx, &storage.GetTeamRequest{ID: team2.ID, IncludeMembers: true})
	require.NoError(t, err)
	recvTeamByURL2, err := client.GetTeam(ctx, &storage.GetTeamRequest{InvitePath: team2.InviteLink, IncludeMembers: true})
	require.NoError(t, err)
	require.Len(t, recvTeamByID2.Members, 1)
	require.Len(t, recvTeamByURL2.Members, 1)
	assert.Equal(t, team2.Captain.ID, recvTeamByID2.Captain.ID)
	assert.Equal(t, team2.Captain.Username, recvTeamByID2.Captain.Username)
	assert.Equal(t, team2.Captain.AvatarURL, recvTeamByID2.Captain.AvatarURL)
	assert.Equal(t, team2.Captain.ID, recvTeamByURL2.Captain.ID)
	assert.Equal(t, team2.Captain.Username, recvTeamByURL2.Captain.Username)
	assert.Equal(t, team2.Captain.AvatarURL, recvTeamByURL2.Captain.AvatarURL)
}

func TestTeamStorage_GetTeam_UserRegistration(t *testing.T) {
	ctx := context.Background()
	client := NewClient(pgtest.NewEmbeddedQuestspaceDB(t))

	user1, err := client.CreateUser(ctx, userReq1)
	require.NoError(t, err)
	user2, err := client.CreateUser(ctx, userReq2)
	require.NoError(t, err)

	qReq1 := questReq1
	qReq1.Creator = user1
	q1, err := client.CreateQuest(ctx, &qReq1)
	require.NoError(t, err)

	tReq1 := teamReq1
	tReq1.Creator = user1
	tReq1.QuestID = q1.ID
	team1, err := client.CreateTeam(ctx, &tReq1)
	require.NoError(t, err)
	require.NoError(t, client.SetInviteLink(ctx, &storage.SetInvitePathRequest{
		TeamID:     team1.ID,
		InvitePath: firstPath,
	}))
	team1.InviteLink = firstPath

	_, err = client.JoinTeam(ctx, &storage.JoinTeamRequest{
		InvitePath: firstPath,
		User:       user2,
	})
	require.NoError(t, err)

	gotTeam, err := client.GetTeam(ctx, &storage.GetTeamRequest{UserRegistration: &storage.UserRegistration{
		UserID:  user2.ID,
		QuestID: q1.ID,
	}})
	require.NoError(t, err)
	assert.NotNil(t, gotTeam)
}

func TestTeamStorage_ChangeLeader(t *testing.T) {
	ctx := context.Background()
	client := NewClient(pgtest.NewEmbeddedQuestspaceDB(t))

	user1, err := client.CreateUser(ctx, userReq1)
	require.NoError(t, err)
	user2, err := client.CreateUser(ctx, userReq2)
	require.NoError(t, err)

	qReq1 := questReq1
	qReq1.Creator = user1
	q1, err := client.CreateQuest(ctx, &qReq1)
	require.NoError(t, err)

	tReq1 := teamReq1
	tReq1.Creator = user1
	tReq1.QuestID = q1.ID
	team1, err := client.CreateTeam(ctx, &tReq1)
	require.NoError(t, err)
	require.NoError(t, client.SetInviteLink(ctx, &storage.SetInvitePathRequest{
		TeamID:     team1.ID,
		InvitePath: firstPath,
	}))
	team1.InviteLink = firstPath

	_, err = client.JoinTeam(ctx, &storage.JoinTeamRequest{InvitePath: firstPath, User: user2})
	require.NoError(t, err)

	_, err = client.ChangeLeader(ctx, &storage.ChangeLeaderRequest{ID: team1.ID, CaptainID: user2.ID})
	require.NoError(t, err)
}

func TestTeamStorage_GetTeam_ErrorOnEmptyRequest(t *testing.T) {
	ctx := context.Background()
	client := NewClient(pgtest.NewEmbeddedQuestspaceDB(t))

	team, err := client.GetTeam(ctx, &storage.GetTeamRequest{})
	require.Error(t, err)
	assert.ErrorIs(t, err, storage.ErrValidation)
	assert.Nil(t, team)
}

func TestTeamStorage_JoinTeam(t *testing.T) {
	ctx := context.Background()
	client := NewClient(pgtest.NewEmbeddedQuestspaceDB(t))

	user1, err := client.CreateUser(ctx, userReq1)
	require.NoError(t, err)
	user2, err := client.CreateUser(ctx, userReq2)
	require.NoError(t, err)

	qReq1 := questReq1
	qReq1.Creator = user1
	q1, err := client.CreateQuest(ctx, &qReq1)
	require.NoError(t, err)

	tReq1 := teamReq1
	tReq1.Creator = user1
	tReq1.QuestID = q1.ID
	team1, err := client.CreateTeam(ctx, &tReq1)
	require.NoError(t, err)
	require.NoError(t, client.SetInviteLink(ctx, &storage.SetInvitePathRequest{
		TeamID:     team1.ID,
		InvitePath: firstPath,
	}))
	team1.InviteLink = firstPath

	invited, err := client.JoinTeam(ctx, &storage.JoinTeamRequest{
		InvitePath: team1.InviteLink,
		User:       &storage.User{ID: user2.ID},
	})
	require.NoError(t, err)
	assert.Equal(t, user2.ID, invited.ID)
	assert.Equal(t, user2.Username, invited.Username)
	assert.Equal(t, user2.AvatarURL, invited.AvatarURL)
}

func TestTeamStorage_JoinTeam_MaxCapacityReached(t *testing.T) {
	ctx := context.Background()
	client := NewClient(pgtest.NewEmbeddedQuestspaceDB(t))

	user1, err := client.CreateUser(ctx, userReq1)
	require.NoError(t, err)
	user2, err := client.CreateUser(ctx, userReq2)
	require.NoError(t, err)

	qReq1 := questReq1
	qReq1.Creator = user1
	qReq1.MaxTeamCap = ptr.Int(1)
	q1, err := client.CreateQuest(ctx, &qReq1)
	require.NoError(t, err)

	tReq1 := teamReq1
	tReq1.Creator = user1
	tReq1.QuestID = q1.ID
	team1, err := client.CreateTeam(ctx, &tReq1)
	require.NoError(t, err)
	require.NoError(t, client.SetInviteLink(ctx, &storage.SetInvitePathRequest{
		TeamID:     team1.ID,
		InvitePath: firstPath,
	}))
	team1.InviteLink = firstPath

	invited, err := client.JoinTeam(ctx, &storage.JoinTeamRequest{
		InvitePath: team1.InviteLink,
		User:       &storage.User{ID: user2.ID},
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, storage.ErrTeamAlreadyFull)
	assert.Nil(t, invited)
}
