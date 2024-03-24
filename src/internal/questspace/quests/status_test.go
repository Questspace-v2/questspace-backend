package quests

import (
	"testing"
	"time"

	"github.com/spkg/ptr"
	"github.com/stretchr/testify/assert"

	"questspace/pkg/storage"
)

var wantNow = time.Date(2024, time.April, 14, 12, 0, 0, 0, time.UTC)

func replaceNowFunc(t *testing.T) {
	nowFunc = func() time.Time {
		return wantNow
	}
	t.Cleanup(func() { nowFunc = time.Now })
}

func TestSetStatus(t *testing.T) {
	replaceNowFunc(t)
	testCases := []struct {
		name           string
		quest          storage.Quest
		expectedStatus storage.QuestStatus
	}{
		{
			name: "finished",
			quest: storage.Quest{
				Status: storage.StatusFinished,
			},
			expectedStatus: storage.StatusFinished,
		},
		{
			name: "registration running with deadline",
			quest: storage.Quest{
				RegistrationDeadline: ptr.Time(wantNow.Add(time.Hour * 4)),
				StartTime:            ptr.Time(wantNow.Add(time.Hour * 28)),
			},
			expectedStatus: storage.StatusOnRegistration,
		},
		{
			name: "registration running without deadline",
			quest: storage.Quest{
				StartTime: ptr.Time(wantNow.Add(time.Hour * 28)),
			},
			expectedStatus: storage.StatusOnRegistration,
		},
		{
			name: "registration done but not started",
			quest: storage.Quest{
				RegistrationDeadline: ptr.Time(wantNow.Add(-time.Hour * 20)),
				StartTime:            ptr.Time(wantNow.Add(time.Hour * 4)),
			},
			expectedStatus: storage.StatusRegistrationDone,
		},
		{
			name: "infinite quest running",
			quest: storage.Quest{
				StartTime: ptr.Time(wantNow.Add(-time.Hour * 24 * 30)),
			},
			expectedStatus: storage.StatusRunning,
		},
		{
			name: "finite quest running",
			quest: storage.Quest{
				StartTime:  ptr.Time(wantNow.Add(-time.Hour * 16)),
				FinishTime: ptr.Time(wantNow.Add(time.Hour * 8)),
			},
			expectedStatus: storage.StatusRunning,
		},
		{
			name: "quest waiting for results after finish time",
			quest: storage.Quest{
				StartTime:  ptr.Time(wantNow.Add(-time.Hour * 28)),
				FinishTime: ptr.Time(wantNow.Add(-time.Hour * 4)),
			},
			expectedStatus: storage.StatusWaitResults,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			q := tc.quest
			SetStatus(&q)
			assert.Equal(t, tc.expectedStatus, q.Status)
		})
	}
}
