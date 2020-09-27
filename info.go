package runner

import (
	"fmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"time"

	"github.com/google/uuid"
)

var defaultOlderThan = time.Hour * 24 * 7

type resultSaver interface {
	SaveTaskResult(TaskResult) error
}

type resultCleaner interface {
	CleanTaskResult(olderThan time.Duration) error
}

type TaskResult struct {
	Name      string
	UID       string
	Err       error
	RawResult []byte
	Started   time.Time
	Finished  time.Time
}

func NewTaskResult() *TaskResult {
	uid := uuid.New().String()
	return &TaskResult{Started: time.Now(), UID: uid}
}

func (r *TaskResult) String() string {
	var result string
	if r.RawResult != nil {
		result = string(r.RawResult)
	}
	return fmt.Sprintf("Job: %s\nUID:%s\nDur:%f\nRes:%s\n\n", r.Name, r.UID, r.Duration().Seconds(), result)
}

func (r *TaskResult) Duration() time.Duration {
	return r.Finished.Sub(r.Started)
}

func ProceedTaskResult(results chan TaskResult, saver resultSaver) {
	for info := range results {
		//save info
		log.Infof("%+v", info)
		if err := saver.SaveTaskResult(info); err != nil {
			err := errors.Wrap(err, "SaveTaskResult")
			log.Error(err)
		}
	}
}

func CleanTaskResult(cleaner resultCleaner) {
	ticker := time.NewTicker(time.Hour)
	for {
		<-ticker.C
		if err := cleaner.CleanTaskResult(defaultOlderThan); err != nil {
			err := errors.Wrap(err, "CleanTaskResult")
			log.Error(err)
		}
	}
}
