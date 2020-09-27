package runner

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"math"
	"time"
)

type TaskStatus int
type TaskRunStrategy string

const (
	TaskStatusIdle     TaskStatus = 1
	TaskStatusDisabled TaskStatus = 2
	TaskStatusRunning  TaskStatus = 3

	TaskViewStatusIdle     = "idle"
	TaskViewStatusDisabled = "disabled"
	TaskViewStatusRunning  = "running"
)

type Task interface {
	Do(context.Context, string) ([]byte, error)
}

type managedTaskResult struct {
	RawResult []byte
	Err       error
}

type managedTask struct {
	name         string
	period       time.Duration
	task         Task
	status       TaskStatus
	lastFinished time.Time
	lastStarted  time.Time
	timeout      time.Duration
}

func newPeriodicTask(name string, period, timeout time.Duration, task Task) *managedTask {
	if period < time.Second {
		period = time.Second
	}
	job := &managedTask{
		name:         name,
		period:       period,
		timeout:      timeout,
		task:         task,
		status:       TaskStatusDisabled,
		lastStarted:  time.Now(),
		lastFinished: time.Now(),
	}
	return job
}

func (t *managedTask) SetStatus(status TaskStatus) {
	t.status = status
}

func (t *managedTask) SetPeriod(period time.Duration) {
	t.period = period
}

func (t *managedTask) SetTimeout(timeout time.Duration) {
	t.timeout = timeout
}

func (t *managedTask) setLastFinished(finished time.Time) {
	t.lastFinished = finished
}

// проверяем сколько времени прошло с момента последнего завершения таска
// если прошло достаточно времени, запускаем
func (t *managedTask) CanExecute() bool {
	timeSinceFinish := time.Since(t.lastFinished)
	return int64(math.Round(float64(timeSinceFinish)/float64(time.Second))) >= int64(t.period/time.Second)
}

func (t *managedTask) Execute(ctx context.Context, uid string, resultCh chan *managedTaskResult) {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("Recovered in ManagedTask.Execute %s(%s): %v", t.name, uid, r)
			resultCh <- &managedTaskResult{
				RawResult: nil,
				Err:       fmt.Errorf("panic %s(%s): %v", t.name, uid, r),
			}
		}
	}()
	t.lastStarted = time.Now()
	log.Infof("Job %s execute..", t.name)
	rawResult, err := t.task.Do(ctx, uid)

	resultCh <- &managedTaskResult{
		RawResult: rawResult,
		Err:       err,
	}
}
