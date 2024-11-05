package pgclient

import (
	"context"
	"testing"
	"time"

	"questspace/internal/pgdb/pgtest"
	"questspace/pkg/storage"

	"github.com/spkg/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetTeamInfos(t *testing.T) {
	ctx := context.Background()
	client := NewClient(pgtest.NewEmbeddedQuestspaceDB(t))

	quest := createTestQuest(t, ctx, client, "svayp11", "quest1")
	tg1, err := client.CreateTaskGroup(ctx, &storage.CreateTaskGroupRequest{
		Name:     "tg1",
		OrderIdx: 0,
		QuestID:  quest.ID,
		PubTime:  ptr.Time(time.Now()),
	})
	require.NoError(t, err)
	tg2, err := client.CreateTaskGroup(ctx, &storage.CreateTaskGroupRequest{
		Name:     "tg2",
		OrderIdx: 1,
		QuestID:  quest.ID,
		PubTime:  ptr.Time(time.Now()),
	})
	require.NoError(t, err)

	user, err := client.CreateUser(ctx, &storage.CreateUserRequest{
		Username:  "sv11",
		Password:  "123",
		AvatarURL: "123",
	})
	require.NoError(t, err)

	team, err := client.CreateTeam(ctx, &storage.CreateTeamRequest{
		Name:               "team",
		QuestID:            quest.ID,
		Creator:            user,
		RegistrationStatus: storage.RegistrationStatusAccepted,
	})
	require.NoError(t, err)

	tgs, err := client.GetTaskGroups(ctx, &storage.GetTaskGroupsRequest{
		QuestID:  quest.ID,
		TeamData: &storage.TeamData{TeamID: &team.ID},
	})
	require.NoError(t, err)

	assert.Equal(t, tg1.ID, tgs[0].ID)
	assert.NotNil(t, tgs[0].TeamInfo)
	assert.Equal(t, tg2.ID, tgs[1].ID)
	assert.Nil(t, tgs[1].TeamInfo)

	now := time.Now().UTC()
	_, err = client.UpsertTeamInfo(ctx, &storage.UpsertTeamInfoRequest{
		TeamID:      team.ID,
		TaskGroupID: tg1.ID,
		OpeningTime: *quest.StartTime,
		ClosingTime: &now,
	})
	require.NoError(t, err)

	tgs, err = client.GetTaskGroups(ctx, &storage.GetTaskGroupsRequest{
		QuestID:  quest.ID,
		TeamData: &storage.TeamData{TeamID: &team.ID},
	})
	require.NoError(t, err)
	assert.NotNil(t, tgs[0].TeamInfo)
	assert.NotNil(t, tgs[1].TeamInfo)
	assert.NotEqual(t, quest.StartTime.Unix(), tgs[1].TeamInfo.OpeningTime.Unix())
	assert.Equal(t, now.Unix(), tgs[1].TeamInfo.OpeningTime.Unix())

	tgs, err = client.GetTaskGroups(ctx, &storage.GetTaskGroupsRequest{
		QuestID:  quest.ID,
		TeamData: &storage.TeamData{UserID: &user.ID},
	})
	require.NoError(t, err)
	assert.NotNil(t, tgs[0].TeamInfo)
	assert.NotNil(t, tgs[1].TeamInfo)

	tg, err := client.GetTaskGroup(ctx, &storage.GetTaskGroupRequest{
		ID:       tg2.ID,
		TeamData: &storage.TeamData{TeamID: &team.ID},
	})
	require.NoError(t, err)
	assert.NotNil(t, tg.TeamInfo)
}
