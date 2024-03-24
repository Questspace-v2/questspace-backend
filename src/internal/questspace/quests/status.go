package quests

import (
	"time"

	"questspace/pkg/storage"
)

var nowFunc = time.Now

func SetStatus(q *storage.Quest) {
	if q.Status == storage.StatusFinished {
		return
	}
	now := nowFunc()
	if q.RegistrationDeadline != nil && q.RegistrationDeadline.After(now) ||
		q.RegistrationDeadline == nil && q.StartTime.After(now) {
		q.Status = storage.StatusOnRegistration
		return
	}
	if q.StartTime.After(now) {
		q.Status = storage.StatusRegistrationDone
		return
	}
	if q.FinishTime == nil || q.FinishTime.After(now) {
		q.Status = storage.StatusRunning
		return
	}
	q.Status = storage.StatusWaitResults
}
