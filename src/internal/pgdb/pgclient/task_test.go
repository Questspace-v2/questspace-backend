package pgclient

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"questspace/internal/pgdb/pgtest"
	"questspace/pkg/storage"
)

var (
	taskReq = storage.CreateTaskRequest{
		Name:     "Why?",
		Question: "Why rlly",
		OrderIdx: 0,
		Reward:   123,
		CorrectAnswers: []string{
			"1", "2", "3",
		},
		Hints: []string{
			"1", "2", "3",
		},
		Verification: storage.VerificationAuto,
		MediaLink:    "https://google.com",
	}
)

func TestTaskStorage_CreateTask(t *testing.T) {
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
}

func TestTaskStorage_GetTask(t *testing.T) {
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

	got, err := client.GetTask(ctx, &storage.GetTaskRequest{ID: task.ID})
	require.NoError(t, err)

	expected := *task
	expected.Group = nil
	assert.Equal(t, expected, *got)
}

func TestTaskStorage_GetAnswerData(t *testing.T) {
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

	got, err := client.GetAnswerData(ctx, &storage.GetTaskRequest{ID: task.ID})
	require.NoError(t, err)

	assert.NotEmpty(t, got.ID)
	assert.Equal(t, task.CorrectAnswers, got.CorrectAnswers)
	assert.Equal(t, task.Reward, got.Reward)
	assert.Equal(t, task.Verification, got.Verification)
	assert.Equal(t, task.Hints, got.Hints)
}

func TestTaskStorage_GetTasks(t *testing.T) {
	ctx := context.Background()
	client := NewClient(pgtest.NewEmbeddedQuestspaceDB(t))

	quest := createTestQuest(t, ctx, client, "svayp11", "quest1")
	tg1, err := client.CreateTaskGroup(ctx, &storage.CreateTaskGroupRequest{
		Name:     "tg1",
		OrderIdx: 0,
		QuestID:  quest.ID,
	})
	require.NoError(t, err)
	tg2, err := client.CreateTaskGroup(ctx, &storage.CreateTaskGroupRequest{
		Name:     "tg2",
		OrderIdx: 1,
		QuestID:  quest.ID,
	})
	require.NoError(t, err)

	taskReq1 := taskReq
	taskReq1.GroupID = tg1.ID
	taskReq1.OrderIdx = 1
	task1, err := client.CreateTask(ctx, &taskReq1)
	require.NoError(t, err)
	taskReq2 := taskReq
	taskReq2.GroupID = tg1.ID
	taskReq2.OrderIdx = 0
	task2, err := client.CreateTask(ctx, &taskReq2)
	require.NoError(t, err)
	taskReq3 := taskReq
	taskReq3.GroupID = tg2.ID
	task3, err := client.CreateTask(ctx, &taskReq3)
	require.NoError(t, err)

	g1Tasks, err := client.GetTasks(ctx, &storage.GetTasksRequest{GroupIDs: []string{tg1.ID}})
	require.NoError(t, err)
	require.Len(t, g1Tasks, 1)
	require.Len(t, g1Tasks[tg1.ID], 2)
	assert.Equal(t, []storage.Task{*task2, *task1}, g1Tasks[tg1.ID])

	g2Tasks, err := client.GetTasks(ctx, &storage.GetTasksRequest{GroupIDs: []string{tg2.ID}})
	require.NoError(t, err)
	require.Len(t, g2Tasks, 1)
	require.Len(t, g2Tasks[tg2.ID], 1)
	assert.Equal(t, []storage.Task{*task3}, g2Tasks[tg2.ID])

	allTasks, err := client.GetTasks(ctx, &storage.GetTasksRequest{QuestID: quest.ID})
	require.NoError(t, err)
	require.Len(t, allTasks, 2)
	require.Len(t, allTasks[tg1.ID], 2)
	require.Len(t, allTasks[tg2.ID], 1)
	assert.Equal(t, []storage.Task{*task2, *task1}, allTasks[tg1.ID])
	assert.Equal(t, []storage.Task{*task3}, allTasks[tg2.ID])
}
