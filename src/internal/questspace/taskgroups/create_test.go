package taskgroups

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"questspace/internal/questspace/taskgroups/requests"
	"questspace/pkg/storage"
	storagemock "questspace/pkg/storage/mocks"
)

func TestService_Create_Basic(t *testing.T) {
	ctrl := gomock.NewController(t)
	s := storagemock.NewMockQuestSpaceStorage(ctrl)
	serv := NewService(s, s, requests.NopValidator{})
	ctx := context.Background()

	req := requests.CreateFullRequest{
		QuestID: "qid",
		TaskGroups: []requests.CreateRequest{
			{
				Name: "1",
				Tasks: []requests.CreateTaskRequest{
					{
						Name:         "1",
						Question:     "1",
						Reward:       1,
						Verification: storage.VerificationAuto,
					},
				},
			},
		},
	}
	createTaskReq := storage.CreateTaskRequest{
		OrderIdx:     0,
		GroupID:      "tg1",
		Name:         req.TaskGroups[0].Tasks[0].Name,
		Question:     req.TaskGroups[0].Tasks[0].Question,
		Reward:       req.TaskGroups[0].Tasks[0].Reward,
		Verification: req.TaskGroups[0].Tasks[0].Verification,
	}
	createReq := &storage.CreateTaskGroupRequest{
		QuestID:  req.QuestID,
		OrderIdx: 0,
		Name:     req.TaskGroups[0].Name,
		Tasks: []storage.CreateTaskRequest{
			createTaskReq,
		},
	}
	// bc we do not know group id before we create it
	createReq.Tasks[0].GroupID = ""

	gomock.InOrder(
		s.EXPECT().GetTaskGroups(ctx, &storage.GetTaskGroupsRequest{QuestID: req.QuestID}).Return(nil, nil).Times(2),

		s.EXPECT().CreateTaskGroup(ctx, createReq).Return(&storage.TaskGroup{
			ID:       "tg1",
			Name:     req.TaskGroups[0].Name,
			OrderIdx: 0,
			Quest:    &storage.Quest{ID: req.QuestID},
		}, nil),

		s.EXPECT().GetTasks(ctx, &storage.GetTasksRequest{GroupIDs: []string{"tg1"}}).Return(nil, nil),

		s.EXPECT().CreateTask(ctx, &createTaskReq).Return(&storage.Task{
			ID:           "t1",
			OrderIdx:     0,
			Group:        &storage.TaskGroup{ID: "tg1"},
			Name:         req.TaskGroups[0].Tasks[0].Name,
			Question:     req.TaskGroups[0].Tasks[0].Question,
			Reward:       req.TaskGroups[0].Tasks[0].Reward,
			Verification: req.TaskGroups[0].Tasks[0].Verification,
		}, nil),

		s.EXPECT().GetTasks(ctx, &storage.GetTasksRequest{GroupIDs: []string{"tg1"}}).Return(nil, nil),

		s.EXPECT().GetTaskGroups(ctx, &storage.GetTaskGroupsRequest{QuestID: req.QuestID, IncludeTasks: true}).Return(nil, nil),
	)

	_, err := serv.Create(ctx, &req)
	require.NoError(t, err)
}
