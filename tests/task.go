package tests

import (
	"context"
	"time"
)

type TestPeriodTask struct {
	RunCount int
	sleep    time.Duration
}

func NewTestPeriodTask(sleep time.Duration) *TestPeriodTask {
	return &TestPeriodTask{sleep: sleep}
}

func (t *TestPeriodTask) Do(context context.Context, test string) ([]byte, error) {
	time.Sleep(t.sleep)
	t.RunCount++
	return nil, nil
}

type TestPeriodPanicTask struct {
	RunCount     int
	sleep        time.Duration
	AlreadyPanic bool
}

func NewTestPeriodPanicTask(sleep time.Duration) *TestPeriodPanicTask {
	return &TestPeriodPanicTask{sleep: sleep}
}

func (t *TestPeriodPanicTask) Do(c context.Context, test string) ([]byte, error) {
	time.Sleep(t.sleep)
	if t.RunCount == 1 && !t.AlreadyPanic {
		t.AlreadyPanic = true
		panic("something going wrong")
	}
	t.RunCount++
	return nil, nil
}
