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

var taskGroupsForTest = []storage.TaskGroup{
	{ID: "1", OrderIdx: 0},
	{ID: "2", OrderIdx: 1},
	{ID: "3", OrderIdx: 2},
	{ID: "4", OrderIdx: 3},
	{ID: "5", OrderIdx: 4},
	{ID: "6", OrderIdx: 5},
	{ID: "7", OrderIdx: 6},
}

func TestUpdater_reorderUpdatedTaskGroups(t *testing.T) {
	testCases := []struct {
		name      string
		initial   taskGroupsPacked
		reqs      []storage.UpdateTaskGroupRequest
		expected  taskGroupsPacked
		expectErr bool
	}{
		{
			name: "no reorders",
			initial: taskGroupsPacked{
				byID: map[storage.ID]*storage.TaskGroup{
					"1": &taskGroupsForTest[0],
					"2": &taskGroupsForTest[1],
					"3": &taskGroupsForTest[2],
				},
				ordered: []*storage.TaskGroup{&taskGroupsForTest[0], &taskGroupsForTest[1], &taskGroupsForTest[2]},
			},
			reqs: []storage.UpdateTaskGroupRequest{{ID: "1", Name: "Changed name"}},
			expected: taskGroupsPacked{
				byID: map[storage.ID]*storage.TaskGroup{
					"1": &taskGroupsForTest[0],
					"2": &taskGroupsForTest[1],
					"3": &taskGroupsForTest[2],
				},
				ordered: []*storage.TaskGroup{&taskGroupsForTest[0], &taskGroupsForTest[1], &taskGroupsForTest[2]},
			},
		},
		{
			name: "one short circular reorder",
			initial: taskGroupsPacked{
				byID: map[storage.ID]*storage.TaskGroup{
					"1": &taskGroupsForTest[0],
					"2": &taskGroupsForTest[1],
					"3": &taskGroupsForTest[2],
				},
				ordered: []*storage.TaskGroup{&taskGroupsForTest[0], &taskGroupsForTest[1], &taskGroupsForTest[2]},
			},
			reqs: []storage.UpdateTaskGroupRequest{{ID: "1", OrderIdx: 1}, {ID: "2", OrderIdx: 0}},
			expected: taskGroupsPacked{
				byID: map[storage.ID]*storage.TaskGroup{
					"1": &taskGroupsForTest[0],
					"2": &taskGroupsForTest[1],
					"3": &taskGroupsForTest[2],
				},
				ordered: []*storage.TaskGroup{&taskGroupsForTest[1], &taskGroupsForTest[0], &taskGroupsForTest[2]},
			},
		},
		{
			name: "one short chain reorder",
			initial: taskGroupsPacked{
				byID: map[storage.ID]*storage.TaskGroup{
					"1": &taskGroupsForTest[0],
					"3": &taskGroupsForTest[2],
				},
				ordered: []*storage.TaskGroup{&taskGroupsForTest[0], nil, &taskGroupsForTest[2]},
			},
			reqs: []storage.UpdateTaskGroupRequest{{ID: "1", OrderIdx: 2}, {ID: "3", OrderIdx: 1}},
			expected: taskGroupsPacked{
				byID: map[storage.ID]*storage.TaskGroup{
					"1": &taskGroupsForTest[0],
					"3": &taskGroupsForTest[2],
				},
				ordered: []*storage.TaskGroup{nil, &taskGroupsForTest[2], &taskGroupsForTest[0]},
			},
		},
		{
			name: "one short and one circular",
			initial: taskGroupsPacked{
				byID: map[storage.ID]*storage.TaskGroup{
					"1": &taskGroupsForTest[0],
					"2": &taskGroupsForTest[1],
					"3": &taskGroupsForTest[2],
					"4": &taskGroupsForTest[3],
					"5": &taskGroupsForTest[4],
				},
				ordered: []*storage.TaskGroup{
					&taskGroupsForTest[0],
					&taskGroupsForTest[1],
					&taskGroupsForTest[2],
					&taskGroupsForTest[3],
					&taskGroupsForTest[4],
					nil,
				},
			},
			reqs: []storage.UpdateTaskGroupRequest{
				{ID: "1", OrderIdx: 2},
				{ID: "3", OrderIdx: 1},
				{ID: "2", OrderIdx: 0},
				{ID: "5", OrderIdx: 3},
				{ID: "4", OrderIdx: 5},
			},
			expected: taskGroupsPacked{
				byID: map[storage.ID]*storage.TaskGroup{
					"1": &taskGroupsForTest[0],
					"2": &taskGroupsForTest[1],
					"3": &taskGroupsForTest[2],
					"4": &taskGroupsForTest[3],
					"5": &taskGroupsForTest[4],
				},
				ordered: []*storage.TaskGroup{
					&taskGroupsForTest[1],
					&taskGroupsForTest[2],
					&taskGroupsForTest[0],
					&taskGroupsForTest[4],
					nil,
					&taskGroupsForTest[3],
				},
			},
		},
		{
			name:      "error on non-existent items",
			reqs:      []storage.UpdateTaskGroupRequest{{ID: "1"}},
			expectErr: true,
		},
		{
			name: "error on more than one items in one index",
			initial: taskGroupsPacked{
				byID: map[storage.ID]*storage.TaskGroup{
					"1": &taskGroupsForTest[0],
					"2": &taskGroupsForTest[1],
					"3": &taskGroupsForTest[2],
				},
				ordered: []*storage.TaskGroup{&taskGroupsForTest[0], &taskGroupsForTest[1], &taskGroupsForTest[2]},
			},
			reqs:      []storage.UpdateTaskGroupRequest{{ID: "1", OrderIdx: 2}, {ID: "2", OrderIdx: 2}},
			expectErr: true,
		},
	}

	updater := NewUpdater(nil, nil, requests.NopValidator{})
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := updater.reorderUpdatedTaskGroups(&tc.initial, tc.reqs)
			if tc.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.expected, tc.initial)
		})
	}
}

func TestUpdater_BulkUpdateTaskGroups_DeleteLastTwo(t *testing.T) {
	ctrl := gomock.NewController(t)
	s := storagemock.NewMockTaskGroupStorage(ctrl)
	updater := NewUpdater(s, nil, requests.NopValidator{})
	ctx := context.Background()

	const questID = "quest-id"
	req := &storage.TaskGroupsBulkUpdateRequest{
		QuestID: questID,
		Delete: []storage.DeleteTaskGroupRequest{
			{ID: "2"},
			{ID: "3"},
		},
	}

	gomock.InOrder(
		s.EXPECT().GetTaskGroups(ctx, &storage.GetTaskGroupsRequest{QuestID: questID}).Return([]storage.TaskGroup{
			taskGroupsForTest[0],
			taskGroupsForTest[1],
			taskGroupsForTest[2],
		}, nil),
		s.EXPECT().DeleteTaskGroup(ctx, &req.Delete[0]).Return(nil),
		s.EXPECT().DeleteTaskGroup(ctx, &req.Delete[1]).Return(nil),
		s.EXPECT().GetTaskGroups(ctx, &storage.GetTaskGroupsRequest{QuestID: questID, IncludeTasks: true}).Return([]storage.TaskGroup{
			taskGroupsForTest[0],
		}, nil),
	)

	_, err := updater.BulkUpdateTaskGroups(ctx, req)
	require.NoError(t, err)
}

func TestUpdater_BulkUpdateTaskGroups_CreateInTheEnd(t *testing.T) {
	ctrl := gomock.NewController(t)
	s := storagemock.NewMockTaskGroupStorage(ctrl)
	updater := NewUpdater(s, nil, requests.NopValidator{})
	ctx := context.Background()

	const questID = "quest-id"
	req := &storage.TaskGroupsBulkUpdateRequest{
		QuestID: questID,
		Create: []storage.CreateTaskGroupRequest{
			{Name: "new group", OrderIdx: 3},
		},
	}
	createdTaskGroup := storage.TaskGroup{ID: "28", Name: "new group", OrderIdx: 3}

	createIDIncluded := req.Create[0]
	createIDIncluded.QuestID = questID
	gomock.InOrder(
		s.EXPECT().GetTaskGroups(ctx, &storage.GetTaskGroupsRequest{QuestID: questID}).Return([]storage.TaskGroup{
			taskGroupsForTest[0],
			taskGroupsForTest[1],
			taskGroupsForTest[2],
		}, nil),
		s.EXPECT().CreateTaskGroup(ctx, &createIDIncluded).Return(&createdTaskGroup, nil),
		s.EXPECT().GetTaskGroups(ctx, &storage.GetTaskGroupsRequest{QuestID: questID, IncludeTasks: true}).Return([]storage.TaskGroup{
			taskGroupsForTest[0],
			taskGroupsForTest[1],
			taskGroupsForTest[2],
			createdTaskGroup,
		}, nil),
	)

	_, err := updater.BulkUpdateTaskGroups(ctx, req)
	require.NoError(t, err)
}

func TestUpdater_BulkUpdateTaskGroups_UpdateWithoutReorder(t *testing.T) {
	ctrl := gomock.NewController(t)
	s := storagemock.NewMockTaskGroupStorage(ctrl)
	updater := NewUpdater(s, nil, requests.NopValidator{})
	ctx := context.Background()

	const questID = "quest-id"
	req := &storage.TaskGroupsBulkUpdateRequest{
		QuestID: questID,
		Update: []storage.UpdateTaskGroupRequest{
			{ID: "1", Name: "New name"},
		},
	}
	changedTaskGroup := storage.TaskGroup{ID: "1", Name: "New name", OrderIdx: 0}

	updateIdIncluded := req.Update[0]
	updateIdIncluded.QuestID = questID
	gomock.InOrder(
		s.EXPECT().GetTaskGroups(ctx, &storage.GetTaskGroupsRequest{QuestID: questID}).Return([]storage.TaskGroup{
			taskGroupsForTest[0],
			taskGroupsForTest[1],
			taskGroupsForTest[2],
		}, nil),
		s.EXPECT().UpdateTaskGroup(ctx, &updateIdIncluded).Return(&changedTaskGroup, nil),
		s.EXPECT().GetTaskGroups(ctx, &storage.GetTaskGroupsRequest{QuestID: questID, IncludeTasks: true}).Return([]storage.TaskGroup{
			changedTaskGroup,
			taskGroupsForTest[1],
			taskGroupsForTest[2],
		}, nil),
	)

	_, err := updater.BulkUpdateTaskGroups(ctx, req)
	require.NoError(t, err)
}

func TestUpdater_BulkUpdateTaskGroups_DeleteWithCreateSubstitution(t *testing.T) {
	ctrl := gomock.NewController(t)
	s := storagemock.NewMockTaskGroupStorage(ctrl)
	updater := NewUpdater(s, nil, requests.NopValidator{})
	ctx := context.Background()

	const questID = "quest-id"
	req := &storage.TaskGroupsBulkUpdateRequest{
		QuestID: questID,
		Delete: []storage.DeleteTaskGroupRequest{
			{ID: "2"},
		},
		Create: []storage.CreateTaskGroupRequest{
			{Name: "new group", OrderIdx: 1},
		},
	}
	createdTaskGroup := storage.TaskGroup{ID: "28", Name: "new group", OrderIdx: 1}

	createIDIncluded := req.Create[0]
	createIDIncluded.QuestID = questID
	gomock.InOrder(
		s.EXPECT().GetTaskGroups(ctx, &storage.GetTaskGroupsRequest{QuestID: questID}).Return([]storage.TaskGroup{
			taskGroupsForTest[0],
			taskGroupsForTest[1],
			taskGroupsForTest[2],
		}, nil),
		s.EXPECT().DeleteTaskGroup(ctx, &req.Delete[0]).Return(nil),
		s.EXPECT().CreateTaskGroup(ctx, &createIDIncluded).Return(&createdTaskGroup, nil),
		s.EXPECT().GetTaskGroups(ctx, &storage.GetTaskGroupsRequest{QuestID: questID, IncludeTasks: true}).Return([]storage.TaskGroup{
			taskGroupsForTest[0],
			createdTaskGroup,
			taskGroupsForTest[2],
		}, nil),
	)

	_, err := updater.BulkUpdateTaskGroups(ctx, req)
	require.NoError(t, err)
}

func TestUpdater_BulkUpdateTaskGroups_DeleteReorderAndCreate(t *testing.T) {
	ctrl := gomock.NewController(t)
	s := storagemock.NewMockTaskGroupStorage(ctrl)
	updater := NewUpdater(s, nil, requests.NopValidator{})
	ctx := context.Background()

	const questID = "quest-id"
	req := &storage.TaskGroupsBulkUpdateRequest{
		QuestID: questID,
		Delete: []storage.DeleteTaskGroupRequest{
			{ID: "2"}, {ID: "6"},
		},
		Update: []storage.UpdateTaskGroupRequest{
			{ID: "1", OrderIdx: 1},
			{ID: "3", OrderIdx: 4},
			{ID: "5", OrderIdx: 3},
			{ID: "4", OrderIdx: 2},
		},
		Create: []storage.CreateTaskGroupRequest{
			{Name: "Replace first", OrderIdx: 0},
		},
	}
	createdTaskGroup := storage.TaskGroup{ID: "28", Name: "Replace first", OrderIdx: 0}
	updatedTaskGroups := []storage.TaskGroup{
		{ID: "1", OrderIdx: 1},
		{ID: "3", OrderIdx: 4},
		{ID: "5", OrderIdx: 3},
		{ID: "4", OrderIdx: 2},
	}

	updateWithIDs := make([]storage.UpdateTaskGroupRequest, len(req.Update))
	copy(updateWithIDs, req.Update)
	for i := range updateWithIDs {
		updateWithIDs[i].QuestID = questID
	}
	createWithID := req.Create[0]
	createWithID.QuestID = questID
	gomock.InOrder(
		s.EXPECT().GetTaskGroups(ctx, &storage.GetTaskGroupsRequest{QuestID: questID}).Return([]storage.TaskGroup{
			taskGroupsForTest[0],
			taskGroupsForTest[1],
			taskGroupsForTest[2],
			taskGroupsForTest[3],
			taskGroupsForTest[4],
			taskGroupsForTest[5],
		}, nil),
		s.EXPECT().DeleteTaskGroup(ctx, &req.Delete[0]).Return(nil),
		s.EXPECT().DeleteTaskGroup(ctx, &req.Delete[1]).Return(nil),

		s.EXPECT().UpdateTaskGroup(ctx, &updateWithIDs[0]).Return(&updatedTaskGroups[0], nil),
		s.EXPECT().UpdateTaskGroup(ctx, &updateWithIDs[1]).Return(&updatedTaskGroups[1], nil),
		s.EXPECT().UpdateTaskGroup(ctx, &updateWithIDs[2]).Return(&updatedTaskGroups[2], nil),
		s.EXPECT().UpdateTaskGroup(ctx, &updateWithIDs[3]).Return(&updatedTaskGroups[3], nil),

		s.EXPECT().CreateTaskGroup(ctx, &createWithID).Return(&createdTaskGroup, nil),
		s.EXPECT().GetTaskGroups(ctx, &storage.GetTaskGroupsRequest{QuestID: questID, IncludeTasks: true}).Return([]storage.TaskGroup{
			// whatever
		}, nil),
	)

	_, err := updater.BulkUpdateTaskGroups(ctx, req)
	require.NoError(t, err)
}
