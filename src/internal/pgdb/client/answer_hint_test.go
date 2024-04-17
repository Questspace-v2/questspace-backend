package pgdb

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"questspace/internal/pgdb/pgtest"
	"questspace/pkg/storage"
)

func createTestTeam(t *testing.T, ctx context.Context, client *Client, quest *storage.Quest, userName, teamName string) (*storage.Team, *storage.User) {
	userReq := storage.CreateUserRequest{
		Username:  userName,
		Password:  "sad",
		AvatarURL: "https://avatars3.githubusercontent.com/u/9878",
	}
	user, err := client.CreateUser(ctx, &userReq)
	require.NoError(t, err)

	teamReq := storage.CreateTeamRequest{
		QuestID: quest.ID,
		Name:    teamName,
		Creator: user,
	}
	team, err := client.CreateTeam(ctx, &teamReq)
	require.NoError(t, err)
	return team, user
}

func TestAnswerHintStorage_TakeHint(t *testing.T) {
	ctx := context.Background()
	client := NewClient(pgtest.NewEmbeddedQuestspaceDB(t))

	quest := createTestQuest(t, ctx, client, "svayp11", "quest1")
	tg, err := client.CreateTaskGroup(ctx, &storage.CreateTaskGroupRequest{
		Name:     "tg1",
		OrderIdx: 0,
		QuestID:  quest.ID,
	})
	require.NoError(t, err)

	taskReq1 := taskReq
	taskReq1.GroupID = tg.ID
	task, err := client.CreateTask(ctx, &taskReq1)
	require.NoError(t, err)
	assert.NotEmpty(t, task.ID)

	team, _ := createTestTeam(t, ctx, client, quest, "svayp22", "team1")
	hintReq := storage.TakeHintRequest{
		TaskID: task.ID,
		TeamID: team.ID,
	}
	for i := range task.Hints {
		hintReq.Index = i
		hint, err := client.TakeHint(ctx, &hintReq)
		require.NoError(t, err)
		assert.Equal(t, hintReq.Index, hint.Index)
		assert.Equal(t, task.Hints[hintReq.Index], hint.Text)
	}
}

func TestAnswerHintStorage_GetHintTakes(t *testing.T) {
	ctx := context.Background()
	client := NewClient(pgtest.NewEmbeddedQuestspaceDB(t))

	quest := createTestQuest(t, ctx, client, "svayp11", "quest1")
	tg, err := client.CreateTaskGroup(ctx, &storage.CreateTaskGroupRequest{
		Name:     "tg1",
		OrderIdx: 0,
		QuestID:  quest.ID,
	})
	require.NoError(t, err)

	taskReq1 := taskReq
	taskReq1.GroupID = tg.ID
	task, err := client.CreateTask(ctx, &taskReq1)
	require.NoError(t, err)
	assert.NotEmpty(t, task.ID)

	team, _ := createTestTeam(t, ctx, client, quest, "svayp22", "team1")
	hintReq := storage.TakeHintRequest{
		TaskID: task.ID,
		TeamID: team.ID,
	}
	for i := range task.Hints {
		hintReq.Index = i
		hint, err := client.TakeHint(ctx, &hintReq)
		require.NoError(t, err)
		assert.Equal(t, hintReq.Index, hint.Index)
		assert.Equal(t, task.Hints[hintReq.Index], hint.Text)
	}

	hintTakes, err := client.GetHintTakes(ctx, &storage.GetHintTakesRequest{TaskID: task.ID, TeamID: team.ID, QuestID: quest.ID})
	require.NoError(t, err)
	require.Len(t, hintTakes[task.ID], len(task.Hints))
	for _, hint := range hintTakes[task.ID] {
		assert.Equal(t, task.Hints[hint.Hint.Index], hint.Hint.Text)
	}
}

func TestAnswerHintStorage_CreateAnswerTry(t *testing.T) {
	ctx := context.Background()
	client := NewClient(pgtest.NewEmbeddedQuestspaceDB(t))

	quest := createTestQuest(t, ctx, client, "svayp11", "quest1")
	tg, err := client.CreateTaskGroup(ctx, &storage.CreateTaskGroupRequest{
		Name:     "tg1",
		OrderIdx: 0,
		QuestID:  quest.ID,
	})
	require.NoError(t, err)

	taskReq1 := taskReq
	taskReq1.GroupID = tg.ID
	task, err := client.CreateTask(ctx, &taskReq1)
	require.NoError(t, err)
	assert.NotEmpty(t, task.ID)

	team, _ := createTestTeam(t, ctx, client, quest, "svayp22", "team1")
	tryReq := storage.CreateAnswerTryRequest{
		Text:     task.CorrectAnswers[0],
		Accepted: true,
		TaskID:   task.ID,
		TeamID:   team.ID,
	}

	require.NoError(t, client.CreateAnswerTry(ctx, &tryReq))
	tasks, err := client.GetAcceptedTasks(ctx, &storage.GetAcceptedTasksRequest{QuestID: quest.ID, TeamID: team.ID})
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	assert.Contains(t, tasks, tryReq.TaskID)
}

func TestAnswerHintStorage_GetScoreResults(t *testing.T) {
	ctx := context.Background()
	client := NewClient(pgtest.NewEmbeddedQuestspaceDB(t))

	quest := createTestQuest(t, ctx, client, "svayp11", "quest1")
	tg, err := client.CreateTaskGroup(ctx, &storage.CreateTaskGroupRequest{
		Name:     "tg1",
		OrderIdx: 0,
		QuestID:  quest.ID,
	})
	require.NoError(t, err)

	taskReq1 := taskReq
	taskReq1.GroupID = tg.ID
	task, err := client.CreateTask(ctx, &taskReq1)
	require.NoError(t, err)
	assert.NotEmpty(t, task.ID)

	team, _ := createTestTeam(t, ctx, client, quest, "svayp22", "team1")
	team2, _ := createTestTeam(t, ctx, client, quest, "svayp222", "team2")
	tryReq := storage.CreateAnswerTryRequest{
		Text:     task.CorrectAnswers[0],
		Accepted: true,
		Score:    task.Reward / 2,
		TaskID:   task.ID,
		TeamID:   team.ID,
	}

	require.NoError(t, client.CreateAnswerTry(ctx, &tryReq))
	results, err := client.GetScoreResults(ctx, &storage.GetResultsRequest{QuestID: quest.ID})
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Len(t, results[team.ID], 1)
	assert.Equal(t, tryReq.Score, results[team.ID][task.ID].Score)
	assert.Nil(t, results[team2.ID])
}
