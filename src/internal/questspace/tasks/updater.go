package tasks

import (
	"context"
	"errors"
	"net/http"
	"slices"

	"github.com/yandex/perforator/library/go/core/xerrors"

	"questspace/internal/questspace/permutations"
	"questspace/pkg/httperrors"
	"questspace/pkg/storage"
)

type Updater struct {
	s storage.TaskStorage
}

func NewUpdater(s storage.TaskStorage) *Updater {
	return &Updater{
		s: s,
	}
}

type tasksPacked struct {
	byID  map[storage.ID]*storage.Task
	order []*storage.Task
}

func fromSlice(s []storage.Task) *tasksPacked {
	byID := make(map[storage.ID]*storage.Task, len(s))
	order := make([]*storage.Task, 0, len(s))
	for _, t := range s {
		t := t
		byID[t.ID] = &t
		order = append(order, &t)
	}
	return &tasksPacked{byID: byID, order: order}
}

func (u *Updater) deleteTasks(ctx context.Context, pack *tasksPacked, deleteReqs []storage.DeleteTaskRequest) error {
	var errs []error
	for _, req := range deleteReqs {
		t, ok := pack.byID[req.ID]
		if !ok {
			errs = append(errs, xerrors.Errorf("not found task %q", req.ID))
			continue
		}

		pack.order[t.OrderIdx] = nil
		delete(pack.byID, req.ID)

		req := req
		if err := u.s.DeleteTask(ctx, &req); err != nil {
			errs = append(errs, xerrors.Errorf("delete task %q: %w", req.ID, err))
		}
	}
	if len(errs) > 0 {
		return xerrors.Errorf("%d error(s) occured during tasks deletion: %w", len(errs), errors.Join(errs...))
	}
	return nil
}

func (u *Updater) getOrderChanges(tasks *tasksPacked, updateReqs []storage.UpdateTaskRequest) ([]permutations.OrderChange, error) {
	reorderTargets := make(map[int]struct{}, len(tasks.order))

	var errs []error
	reorders := make([]permutations.OrderChange, 0, len(updateReqs))
	for _, updateReq := range updateReqs {
		task, ok := tasks.byID[updateReq.ID]
		if !ok {
			errs = append(errs, xerrors.Errorf("not found task %s", updateReq.ID))
			continue
		}
		if task.OrderIdx == updateReq.OrderIdx {
			continue
		}

		if _, used := reorderTargets[updateReq.OrderIdx]; used {
			errs = append(errs, xerrors.Errorf("two or more tasks replacement into one index %d", updateReq.OrderIdx))
			continue
		}
		reorderTargets[updateReq.OrderIdx] = struct{}{}

		reorders = append(reorders, permutations.OrderChange{Prev: task.OrderIdx, Next: updateReq.OrderIdx})
	}
	if len(errs) > 0 {
		return nil, xerrors.Errorf("%d error(s) occured during reorder search: %w", len(errs), errors.Join(errs...))
	}

	return reorders, nil
}

func (u *Updater) reorderUpdatedTasks(tasks *tasksPacked, updateReqs []storage.UpdateTaskRequest) error {
	reorders, err := u.getOrderChanges(tasks, updateReqs)
	if err != nil {
		return httperrors.WrapWithCode(http.StatusBadRequest, err)
	}
	if len(reorders) == 0 {
		return nil
	}

	var errs []error
	trees, cycles := permutations.FindTreesAndCycles(reorders, len(tasks.order))
	for _, tree := range trees {
		if idx := tree[len(tree)-1]; tasks.order[idx] != nil {
			errs = append(errs, xerrors.Errorf("cannot replace item to already taken index %d", idx))
			continue
		}
		prev := tasks.order[tree[0]]
		tasks.order[tree[0]] = nil
		for _, nextIdx := range tree[1:] {
			prev, tasks.order[nextIdx] = tasks.order[nextIdx], prev
		}
	}
	if len(errs) > 0 {
		return httperrors.WrapWithCode(http.StatusBadRequest, errors.Join(errs...))
	}

	for _, cycle := range cycles {
		prev := tasks.order[cycle[0]]
		tasks.order[cycle[0]] = tasks.order[cycle[len(cycle)-1]]
		for _, nextIdx := range cycle[1:] {
			prev, tasks.order[nextIdx] = tasks.order[nextIdx], prev
		}
	}

	return nil
}

func (u *Updater) validatePenalties(score int, hints []storage.Hint) error {
	totalPenalty := 0
	for _, h := range hints {
		totalPenalty += h.Penalty.GetPenaltyPoints(score)
	}
	if totalPenalty > score {
		return httperrors.Errorf(http.StatusBadRequest, "total hints penalty should not exceed maximum score: score %d vs penalty %d", score, totalPenalty)
	}
	return nil
}

func (u *Updater) updateTasks(ctx context.Context, tasks *tasksPacked, updateReqs []storage.UpdateTaskRequest) error {
	var errs []error
	for _, updateReq := range updateReqs {
		updateReq := updateReq
		task, err := u.s.UpdateTask(ctx, &updateReq)
		if err != nil {
			errs = append(errs, xerrors.Errorf("update task %q: %w", updateReq.ID, err))
			continue
		}
		if err = u.validatePenalties(task.Reward, task.FullHints); err != nil {
			errs = append(errs, xerrors.Errorf("task %q: %w", task.ID, err))
		}
		tasks.byID[task.ID] = task
		tasks.order[task.OrderIdx] = task
	}
	if len(errs) > 0 {
		return xerrors.Errorf("%d error(s) occured during tasks update: %w", len(errs), errors.Join(errs...))
	}
	return nil
}

func (u *Updater) createTasks(ctx context.Context, tasks *tasksPacked, createReqs []storage.CreateTaskRequest, groupID storage.ID) error {
	var errs []error
	for _, createReq := range createReqs {
		if tasks.order[createReq.OrderIdx] != nil {
			errs = append(errs, xerrors.Errorf("cannot create task group with name %q: %d index already taken", createReq.Name, createReq.OrderIdx))
			continue
		}
	}
	if len(errs) > 0 {
		return httperrors.WrapWithCode(http.StatusBadRequest, errors.Join(errs...))
	}

	for _, createReq := range createReqs {
		createReq := createReq
		createReq.GroupID = groupID
		task, err := u.s.CreateTask(ctx, &createReq)
		if err != nil {
			errs = append(errs, xerrors.Errorf("create task group: %w", err))
			continue
		}
		if err = u.validatePenalties(task.Reward, task.FullHints); err != nil {
			errs = append(errs, xerrors.Errorf("task %q: %w", task.ID, err))
		}
		tasks.byID[task.ID] = task
		tasks.order[task.OrderIdx] = task
	}
	if len(errs) > 0 {
		return xerrors.Errorf("%d error(s) occured during task groups create: %w", len(errs), errors.Join(errs...))
	}
	return nil
}

// BulkUpdate
// TODO: unit-tests
func (u *Updater) BulkUpdate(ctx context.Context, req *storage.TasksBulkUpdateRequest) ([]storage.Task, error) {
	oldTasks, err := u.s.GetTasks(ctx, &storage.GetTasksRequest{GroupIDs: []storage.ID{req.GroupID}})
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, httperrors.Errorf(http.StatusNotFound, "quest %q not found", req.QuestID)
		}
		return nil, xerrors.Errorf("get old tasks: %w", err)
	}
	oldTasksSlice := oldTasks[req.GroupID]
	pack := fromSlice(oldTasksSlice)
	if err := u.deleteTasks(ctx, pack, req.Delete); err != nil {
		return nil, xerrors.Errorf("delete tasks: %w", err)
	}
	newLen := len(oldTasksSlice) + len(req.Create) - len(req.Delete)
	if newLen > len(oldTasksSlice) {
		pack.order = slices.Grow(pack.order, newLen-len(oldTasksSlice))
		pack.order = pack.order[:newLen]
	}
	if err := u.reorderUpdatedTasks(pack, req.Update); err != nil {
		return nil, xerrors.Errorf("reorder tasks: %w", err)
	}
	if err := u.updateTasks(ctx, pack, req.Update); err != nil {
		return nil, xerrors.Errorf("update tasks: %w", err)
	}
	if err := u.createTasks(ctx, pack, req.Create, req.GroupID); err != nil {
		return nil, xerrors.Errorf("create tasks: %w", err)
	}

	if err = u.clapPack(ctx, pack); err != nil {
		return nil, xerrors.Errorf("clap tasks pack: %w", err)
	}
	pack.order = pack.order[:newLen]

	newTasks, err := u.s.GetTasks(ctx, &storage.GetTasksRequest{GroupIDs: []storage.ID{req.GroupID}})
	if err != nil {
		return nil, xerrors.Errorf("get new tasks: %w", err)
	}
	return newTasks[req.GroupID], nil
}

func (u *Updater) clapPack(ctx context.Context, pack *tasksPacked) error {
	var errs []error
	var l, r int
	for r != len(pack.order) {
		if pack.order[r] == nil {
			r++
			continue
		}
		pack.order[l], pack.order[r] = pack.order[r], pack.order[l]

		t := pack.order[l]
		if t.OrderIdx != l {
			// TODO(svayp11): Do clapping before database update to reduce number of roundtrips
			newTask, err := u.s.UpdateTask(ctx, &storage.UpdateTaskRequest{ID: t.ID, OrderIdx: l})
			if err != nil {
				errs = append(errs, err)
			}
			pack.order[l] = newTask
		}

		r++
		l++
	}
	if len(errs) > 0 {
		return xerrors.Errorf("%w", errors.Join(errs...))
	}

	return nil
}
