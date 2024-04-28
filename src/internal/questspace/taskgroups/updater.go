package taskgroups

import (
	"context"
	"errors"
	"net/http"
	"slices"

	"golang.org/x/xerrors"

	"questspace/internal/questspace/permutations"
	"questspace/internal/questspace/tasks"
	"questspace/pkg/httperrors"
	"questspace/pkg/storage"
)

type Updater struct {
	s           storage.TaskGroupStorage
	taskUpdater *tasks.Updater
}

type taskGroupsPacked struct {
	byID    map[string]*storage.TaskGroup
	ordered []*storage.TaskGroup
}

func NewUpdater(s storage.TaskGroupStorage, taskUpdater *tasks.Updater) *Updater {
	return &Updater{
		s:           s,
		taskUpdater: taskUpdater,
	}
}

func (u *Updater) getOldTaskGroups(ctx context.Context, questID string) (*taskGroupsPacked, error) {
	oldTaskGroups, err := u.s.GetTaskGroups(ctx, &storage.GetTaskGroupsRequest{QuestID: questID})
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, storage.ErrNotFound
		}
		return nil, xerrors.Errorf("read task groups: %w", err)
	}

	taskGroupIDMap := make(map[string]*storage.TaskGroup, len(oldTaskGroups))
	ordered := make([]*storage.TaskGroup, 0, len(oldTaskGroups))
	for _, tg := range oldTaskGroups {
		tg := tg
		taskGroupIDMap[tg.ID] = &tg
		ordered = append(ordered, &tg)
	}

	return &taskGroupsPacked{
		byID:    taskGroupIDMap,
		ordered: ordered,
	}, nil
}

func (u *Updater) deleteTaskGroups(ctx context.Context, taskGroups *taskGroupsPacked, deleteReqs []storage.DeleteTaskGroupRequest) error {
	var errs []error
	for _, deleteReq := range deleteReqs {
		taskGroup, ok := taskGroups.byID[deleteReq.ID]
		if !ok {
			errs = append(errs, xerrors.Errorf("not found task group %q in quest gruop", deleteReq.ID))
			continue
		}
		taskGroups.ordered[taskGroup.OrderIdx] = nil
		delete(taskGroups.byID, taskGroup.ID)
	}
	if len(errs) > 0 {
		return httperrors.WrapWithCode(http.StatusBadRequest, errors.Join(errs...))
	}

	for _, deleteReq := range deleteReqs {
		deleteReq := deleteReq
		if err := u.s.DeleteTaskGroup(ctx, &deleteReq); err != nil {
			errs = append(errs, xerrors.Errorf("delete task group %s: %w", deleteReq.ID, err))
		}
	}
	if len(errs) > 0 {
		return xerrors.Errorf("%d error(s) occured during task groups delete: %w", len(errs), errors.Join(errs...))
	}
	return nil
}

func (u *Updater) getOrderChanges(taskGroups *taskGroupsPacked, updateReqs []storage.UpdateTaskGroupRequest) ([]permutations.OrderChange, error) {
	reorderTargets := make(map[int]struct{}, len(taskGroups.ordered))

	var errs []error
	reorders := make([]permutations.OrderChange, 0, len(updateReqs))
	for _, updateReq := range updateReqs {
		taskGroup, ok := taskGroups.byID[updateReq.ID]
		if !ok {
			errs = append(errs, xerrors.Errorf("not found task group %s", updateReq.ID))
			continue
		}
		if taskGroup.OrderIdx == updateReq.OrderIdx {
			continue
		}

		if _, used := reorderTargets[updateReq.OrderIdx]; used {
			errs = append(errs, xerrors.Errorf("two or more task groups replacement into one index %d", updateReq.OrderIdx))
			continue
		}
		reorderTargets[updateReq.OrderIdx] = struct{}{}

		reorders = append(reorders, permutations.OrderChange{Prev: taskGroup.OrderIdx, Next: updateReq.OrderIdx})
	}
	if len(errs) > 0 {
		return nil, xerrors.Errorf("%d error(s) occured during reorder search: %w", len(errs), errors.Join(errs...))
	}

	return reorders, nil
}

func (u *Updater) reorderUpdatedTaskGroups(taskGroups *taskGroupsPacked, updateReqs []storage.UpdateTaskGroupRequest) error {
	reorders, err := u.getOrderChanges(taskGroups, updateReqs)
	if err != nil {
		return httperrors.WrapWithCode(http.StatusBadRequest, err)
	}
	if len(reorders) == 0 {
		return nil
	}

	var errs []error
	trees, cycles := permutations.FindTreesAndCycles(reorders, len(taskGroups.ordered))
	for _, tree := range trees {
		if idx := tree[len(tree)-1]; taskGroups.ordered[idx] != nil {
			errs = append(errs, xerrors.Errorf("cannot replace item to already taken index %d", idx))
			continue
		}
		prev := taskGroups.ordered[tree[0]]
		taskGroups.ordered[tree[0]] = nil
		for _, nextIdx := range tree[1:] {
			prev, taskGroups.ordered[nextIdx] = taskGroups.ordered[nextIdx], prev
		}
	}
	if len(errs) > 0 {
		return httperrors.WrapWithCode(http.StatusBadRequest, errors.Join(errs...))
	}

	for _, cycle := range cycles {
		prev := taskGroups.ordered[cycle[0]]
		taskGroups.ordered[cycle[0]] = taskGroups.ordered[cycle[len(cycle)-1]]
		for _, nextIdx := range cycle[1:] {
			prev, taskGroups.ordered[nextIdx] = taskGroups.ordered[nextIdx], prev
		}
	}

	return nil
}

func (u *Updater) updateTaskGroups(ctx context.Context, taskGroups *taskGroupsPacked, updateReqs []storage.UpdateTaskGroupRequest, questID string) error {
	var errs []error
	for _, updateReq := range updateReqs {
		updateReq := updateReq
		updateReq.QuestID = questID
		taskGroup, err := u.s.UpdateTaskGroup(ctx, &updateReq)
		if err != nil {
			errs = append(errs, xerrors.Errorf("update task group %s: %w", updateReq.ID, err))
			continue
		}
		taskGroups.byID[taskGroup.ID] = taskGroup
		taskGroups.ordered[taskGroup.OrderIdx] = taskGroup
		if updateReq.Tasks != nil {
			updateReq.Tasks.GroupID = updateReq.ID
			updateReq.Tasks.QuestID = questID
			taskGroup.Tasks, err = u.taskUpdater.BulkUpdate(ctx, updateReq.Tasks)
			if err != nil {
				errs = append(errs, xerrors.Errorf("update tasks for group %q: %w", updateReq.ID, err))
			}
		}
	}
	if len(errs) > 0 {
		return xerrors.Errorf("%d error(s) occured during task groups update: %w", len(errs), errors.Join(errs...))
	}
	return nil
}

func (u *Updater) createTaskGroups(ctx context.Context, taskGroups *taskGroupsPacked, createReqs []storage.CreateTaskGroupRequest, questID string) error {
	var errs []error
	for _, createReq := range createReqs {
		if taskGroups.ordered[createReq.OrderIdx] != nil {
			errs = append(errs, xerrors.Errorf("cannot create task group with name %q: %d index already taken", createReq.Name, createReq.OrderIdx))
			continue
		}
	}
	if len(errs) > 0 {
		return httperrors.WrapWithCode(http.StatusBadRequest, errors.Join(errs...))
	}

	for _, createReq := range createReqs {
		createReq := createReq
		createReq.QuestID = questID
		taskGroup, err := u.s.CreateTaskGroup(ctx, &createReq)
		if err != nil {
			errs = append(errs, xerrors.Errorf("create task group: %w", err))
			continue
		}
		taskGroups.byID[taskGroup.ID] = taskGroup
		taskGroups.ordered[taskGroup.OrderIdx] = taskGroup
		if createReq.Tasks != nil {
			taskGroup.Tasks, err = u.taskUpdater.BulkUpdate(ctx, &storage.TasksBulkUpdateRequest{
				QuestID: questID,
				GroupID: taskGroup.ID,
				Create:  createReq.Tasks,
			})
			if err != nil {
				errs = append(errs, xerrors.Errorf("create tasks for group %q: %w", taskGroup.ID, err))
			}
		}
	}
	if len(errs) > 0 {
		return xerrors.Errorf("%d error(s) occured during task groups create: %w", len(errs), errors.Join(errs...))
	}
	return nil
}

func (u *Updater) BulkUpdateTaskGroups(ctx context.Context, req *storage.TaskGroupsBulkUpdateRequest) ([]storage.TaskGroup, error) {
	taskGroups, err := u.getOldTaskGroups(ctx, req.QuestID)
	if err != nil {
		return nil, xerrors.Errorf("get old task groups: %w", err)
	}
	if err := u.deleteTaskGroups(ctx, taskGroups, req.Delete); err != nil {
		return nil, xerrors.Errorf("delete task groups from quest %s: %w", req.QuestID, err)
	}
	newLen := len(taskGroups.ordered) + len(req.Create) - len(req.Delete)
	if newLen > len(taskGroups.ordered) {
		taskGroups.ordered = slices.Grow(taskGroups.ordered, newLen-len(taskGroups.ordered))
		taskGroups.ordered = taskGroups.ordered[:newLen]
	}
	if err := u.reorderUpdatedTaskGroups(taskGroups, req.Update); err != nil {
		return nil, xerrors.Errorf("reorder updated task groups: %w", err)
	}
	if err := u.updateTaskGroups(ctx, taskGroups, req.Update, req.QuestID); err != nil {
		return nil, xerrors.Errorf("%w", err)
	}
	if err := u.createTaskGroups(ctx, taskGroups, req.Create, req.QuestID); err != nil {
		return nil, xerrors.Errorf("%w", err)
	}
	taskGroups.ordered = taskGroups.ordered[:newLen]
	var errs []error
	for i, item := range taskGroups.ordered {
		if item == nil {
			errs = append(errs, xerrors.Errorf("index %d is empty", i))
		}
	}
	if len(errs) > 0 {
		return nil, httperrors.WrapWithCode(http.StatusBadRequest, errors.Join(errs...))
	}

	newTaskGroups, err := u.s.GetTaskGroups(ctx, &storage.GetTaskGroupsRequest{QuestID: req.QuestID, IncludeTasks: true})
	if err != nil {
		return nil, xerrors.Errorf("get all task groups: %w", err)
	}
	return newTaskGroups, nil
}
