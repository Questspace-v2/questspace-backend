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

func createTestQuest(t *testing.T, ctx context.Context, client *Client, username, questname string) *storage.Quest {
	user, err := client.CreateUser(ctx, &storage.CreateUserRequest{
		Username:  username,
		Password:  "some_password",
		AvatarURL: "https://ya.ru",
	})
	require.NoError(t, err)
	now := time.Now().UTC()
	quest, err := client.CreateQuest(ctx, &storage.CreateQuestRequest{
		Name:                 questname,
		Description:          "desc",
		Access:               storage.AccessPublic,
		Creator:              user,
		RegistrationDeadline: &now,
		StartTime:            &now,
		FinishTime:           &now,
		MediaLink:            "https://github.com",
		MaxTeamCap:           ptr.Int(3),
	})
	require.NoError(t, err)
	return quest
}

func TestTaskGroupStorage_CreateTaskGroup(t *testing.T) {
	ctx := context.Background()
	client := NewClient(pgtest.NewEmbeddedQuestspaceDB(t))

	quest := createTestQuest(t, ctx, client, "svayp11", "quest1")
	tg, err := client.CreateTaskGroup(ctx, &storage.CreateTaskGroupRequest{
		Name:     "tg1",
		OrderIdx: 0,
		QuestID:  quest.ID,
		PubTime:  ptr.Time(time.Now()),
	})
	require.NoError(t, err)
	assert.Equal(t, 0, tg.OrderIdx)
}

func TestTaskGroupStorage_CreateTaskGroup_Tx(t *testing.T) {
	ctx := context.Background()
	cl, tx := pgtest.NewEmbeddedQuestspaceTx(t)
	client := NewClient(cl)

	quest := createTestQuest(t, ctx, client, "svayp11", "quest1")
	tg, err := client.CreateTaskGroup(ctx, &storage.CreateTaskGroupRequest{
		Name:     "tg1",
		OrderIdx: 0,
		QuestID:  quest.ID,
		PubTime:  ptr.Time(time.Now()),
	})
	require.NoError(t, err)
	assert.Equal(t, 0, tg.OrderIdx)

	tg1, err := client.CreateTaskGroup(ctx, &storage.CreateTaskGroupRequest{
		Name:     "tg2",
		OrderIdx: 0,
		QuestID:  quest.ID,
		PubTime:  ptr.Time(time.Now()),
	})
	require.NoError(t, err)
	assert.Equal(t, 0, tg1.OrderIdx)

	require.Error(t, tx.Commit())
}

func TestTaskGroupStorage_CreateTaskGroup_FailsOnIdenticalOrder(t *testing.T) {
	ctx := context.Background()
	client := NewClient(pgtest.NewEmbeddedQuestspaceDB(t))

	quest := createTestQuest(t, ctx, client, "svayp11", "quest1")
	tg, err := client.CreateTaskGroup(ctx, &storage.CreateTaskGroupRequest{
		Name:     "tg1",
		OrderIdx: 0,
		QuestID:  quest.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, 0, tg.OrderIdx)
	tg2, err := client.CreateTaskGroup(ctx, &storage.CreateTaskGroupRequest{
		Name:     "tg2",
		OrderIdx: 0,
		QuestID:  quest.ID,
	})
	require.Error(t, err)
	assert.Nil(t, tg2)
}

func TestTaskGroupStorage_UpdateTaskGroup_NoErrorOnReorderTx(t *testing.T) {
	ctx := context.Background()
	db, tx := pgtest.NewEmbeddedQuestspaceTx(t)
	client := NewClient(db)

	quest := createTestQuest(t, ctx, client, "svayp11", "quest1")
	tg, err := client.CreateTaskGroup(ctx, &storage.CreateTaskGroupRequest{
		Name:     "tg1",
		OrderIdx: 0,
		QuestID:  quest.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, 0, tg.OrderIdx)
	tg2, err := client.CreateTaskGroup(ctx, &storage.CreateTaskGroupRequest{
		Name:     "tg2",
		OrderIdx: 1,
		QuestID:  quest.ID,
	})
	require.NoError(t, err)

	rTg1, err := client.UpdateTaskGroup(ctx, &storage.UpdateTaskGroupRequest{
		QuestID:  quest.ID,
		ID:       tg.ID,
		OrderIdx: 1,
	})
	require.NoError(t, err)
	rTg2, err := client.UpdateTaskGroup(ctx, &storage.UpdateTaskGroupRequest{
		QuestID:  quest.ID,
		ID:       tg2.ID,
		OrderIdx: 0,
	})
	require.NoError(t, err)
	require.NoError(t, tx.Commit())
	assert.Equal(t, 1, rTg1.OrderIdx)
	assert.Equal(t, 0, rTg2.OrderIdx)
	assert.Equal(t, tg.Name, rTg1.Name)
	assert.Equal(t, tg2.Name, rTg2.Name)
}
