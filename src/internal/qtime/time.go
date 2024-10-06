package qtime

import (
	"os"
	"testing"
	"time"
)

const testEnv = "TIME_TEST"

var (
	nowFunc       = time.Now
	testStartTime = time.Date(2024, time.April, 7, 12, 0, 0, 0, time.UTC)
)

func init() {
	if IsTestTimeMode() {
		nowFunc = func() time.Time {
			return testStartTime
		}
	}
}

func IsTestTimeMode() bool {
	return len(os.Getenv(testEnv)) > 0
}

type TimeGetter func() time.Time

func Now() time.Time {
	return nowFunc()
}

func SetNowFunc(t *testing.T, f TimeGetter) {
	oldFunc := nowFunc
	t.Cleanup(func() {
		nowFunc = oldFunc
	})
	nowFunc = f
}

func Wait(d time.Duration) {
	if IsTestTimeMode() {
		panic("waiting in production environment")
	}
	newTime := testStartTime.Add(d)
	nowFunc = func() time.Time {
		return newTime
	}
}
