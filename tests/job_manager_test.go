package tests

import (
	"context"
	runner "github.com/0ndi/go-task-manager"
	"github.com/0ndi/go-task-manager/models"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"testing"
	"time"
)

const (
	periodJobName  = "test_period"
	periodJobName2 = "test_period2"
)

func TestJobManager_StartStoreInitConfig(t *testing.T) {
	settingsManager := NewSettingsManager()
	task := NewTestPeriodTask(0)

	results := make(chan runner.TaskResult)
	go func() {
		for {
			t := <-results
			log.Info(t.String())
		}
	}()

	taskManager := runner.GetTaskManager(settingsManager, results)
	taskManager.AddTask(periodJobName, time.Minute*2, time.Minute*2, task)
	taskManager.SetWaitTime(time.Second)

	taskManager.Start()
	time.Sleep(time.Second)
	taskManager.Stop()

	log.Infof("%+v", TaskData)
	storedTask, err := settingsManager.GetSettings(periodJobName)
	if err != nil {
		err := errors.Wrap(err, "GetSettings")
		t.Fatal(err)
	}
	log.Infof("stored job: %+v", storedTask)

	if runner.TaskStatus(storedTask.Status) != runner.TaskStatusDisabled {
		t.Errorf("wrong job status: %d", storedTask.Status)
	}

	if storedTask.Name != periodJobName {
		t.Errorf("wrong name %s", storedTask.Name)
	}

	if storedTask.PeriodInSeconds != 120 {
		t.Errorf("wrong period: %d", storedTask.PeriodInSeconds)
	}
	if storedTask.TimeoutInSeconds != 120 {
		t.Errorf("wrong timeout: %d", storedTask.TimeoutInSeconds)
	}

	if task.RunCount != 0 {
		t.Errorf("task should not run")
	}
}

func TestJobManager_Start_PeriodTask(t *testing.T) {
	settingsManager := NewSettingsManager()
	task := NewTestPeriodTask(0)
	task2 := NewTestPeriodTask(0)

	results := make(chan runner.TaskResult)
	go func() {
		for {
			t := <-results
			log.Info(t.String())
		}
	}()

	taskManager := runner.GetTaskManager(settingsManager, results)
	taskManager.SetWaitTime(time.Second*2).
		AddTask(periodJobName, time.Second*5, time.Second, task).
		AddTask(periodJobName2, time.Second*2, time.Second, task2)
	taskManager.Start()

	taskSettings := models.TaskSettings{
		Name:             periodJobName,
		PeriodInSeconds:  1,
		TimeoutInSeconds: 1,
		Status:           int(runner.TaskStatusIdle),
		LastFinished:     time.Now().Add(-time.Minute), // хотим запуск на первом тике
	}
	if err := settingsManager.UpdateSettings(taskSettings); err != nil {
		err := errors.Wrap(err, "UpdateSettings")
		t.Fatal(err)
	}

	taskSettings2 := models.TaskSettings{
		Name:             periodJobName2,
		PeriodInSeconds:  2,
		TimeoutInSeconds: 1,
		Status:           int(runner.TaskStatusIdle),
		LastFinished:     time.Now().Add(-time.Minute), // хотим запуск на первом тике
	}
	if err := settingsManager.UpdateSettings(taskSettings2); err != nil {
		err := errors.Wrap(err, "UpdateSettings")
		t.Fatal(err)
	}
	taskManager.Reload()

	time.Sleep(time.Second * 4)
	taskManager.Stop()

	if task.RunCount != 4 {
		t.Errorf("wrong run count: %d", task.RunCount)
	}
	if task2.RunCount != 2 {
		t.Errorf("wrong run count: %d", task2.RunCount)
	}

}

func TestJobManager_Start_Shutdown(t *testing.T) {
	settingsManager := NewSettingsManager()

	task := NewTestPeriodTask(time.Second)
	task2 := NewTestPeriodTask(time.Second)

	results := make(chan runner.TaskResult)
	go func() {
		for {
			t := <-results
			log.Info(t.String())
		}
	}()

	taskManager := runner.GetTaskManager(settingsManager, results)
	taskManager.SetWaitTime(time.Second*2).
		AddTask(periodJobName, time.Second*5, time.Second*5, task).
		AddTask(periodJobName2, time.Second*2, time.Second*5, task2)

	taskSettings := models.TaskSettings{
		Name:             periodJobName,
		PeriodInSeconds:  1,
		TimeoutInSeconds: 5,
		Status:           int(runner.TaskStatusIdle),
		LastFinished:     time.Now().Add(-time.Minute), // хотим запуск на первом тике
	}
	if err := settingsManager.CreateSettings(taskSettings); err != nil {
		err := errors.Wrap(err, "CreateSettings")
		t.Fatal(err)
	}

	taskSettings2 := models.TaskSettings{
		Name:             periodJobName2,
		PeriodInSeconds:  2,
		TimeoutInSeconds: 5,
		Status:           int(runner.TaskStatusIdle),
		LastFinished:     time.Now().Add(-time.Minute), // хотим запуск на первом тике
	}
	if err := settingsManager.CreateSettings(taskSettings2); err != nil {
		err := errors.Wrap(err, "CreateSettings")
		t.Fatal(err)
	}

	taskManager.Start()
	time.Sleep(time.Second * 4)
	taskManager.Stop()
	time.Sleep(time.Second)

	if task.RunCount != 2 {
		t.Errorf("wrong run count: %d", task.RunCount)
	}
	if task2.RunCount != 1 {
		t.Errorf("wrong run count: %d", task2.RunCount)
	}
}

func TestJobManager_Start_LoadConfig(t *testing.T) {
	settingsManager := NewSettingsManager()
	task := NewTestPeriodTask(0)
	task2 := NewTestPeriodTask(0)

	results := make(chan runner.TaskResult)
	go func() {
		for {
			t := <-results
			log.Info(t.String())
		}
	}()

	taskManager := runner.GetTaskManager(settingsManager, results)
	taskManager.SetWaitTime(time.Second*2).
		AddTask(periodJobName, time.Second*5, time.Second, task).
		AddTask(periodJobName2, time.Second*2, time.Second, task2)

	taskSettings := models.TaskSettings{
		Name:             periodJobName,
		PeriodInSeconds:  1,
		TimeoutInSeconds: 1,
		Status:           int(runner.TaskStatusIdle),
		LastFinished:     time.Now().Add(-time.Minute), // хотим запуск на первом тике
	}
	if err := settingsManager.CreateSettings(taskSettings); err != nil {
		err := errors.Wrap(err, "CreateSettings")
		t.Fatal(err)
	}

	taskSettings2 := models.TaskSettings{
		Name:             periodJobName2,
		PeriodInSeconds:  2,
		TimeoutInSeconds: 1,
		Status:           int(runner.TaskStatusIdle),
		LastFinished:     time.Now().Add(-time.Minute), // хотим запуск на первом тике
	}
	if err := settingsManager.CreateSettings(taskSettings2); err != nil {
		err := errors.Wrap(err, "CreateSettings")
		t.Fatal(err)
	}

	taskManager.Start()
	time.Sleep(time.Second * 4)
	taskManager.Stop()

	if task.RunCount != 4 {
		t.Errorf("wrong run count: %d", task.RunCount)
	}
	if task2.RunCount != 2 {
		t.Errorf("wrong run count: %d", task2.RunCount)
	}
}

func TestJobManager_Start_Panic(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode. Because it takes about 5 minutes to run")
	}
	settingsManager := NewSettingsManager()
	task := NewTestPeriodTask(0)
	task2 := NewTestPeriodPanicTask(0)

	results := make(chan runner.TaskResult)
	go func() {
		for {
			t := <-results
			log.Info(t.String())
		}
	}()

	taskManager := runner.GetTaskManager(settingsManager, results)
	taskManager.SetWaitTime(time.Second*2).
		AddTask(periodJobName, time.Second*5, time.Second, task).
		AddTask(periodJobName2, time.Second*2, time.Second, task2)

	taskSettings := models.TaskSettings{
		Name:             periodJobName,
		PeriodInSeconds:  1,
		TimeoutInSeconds: 1,
		Status:           int(runner.TaskStatusIdle),
		LastFinished:     time.Now().Add(-time.Minute), // хотим запуск на первом тике
	}
	if err := settingsManager.CreateSettings(taskSettings); err != nil {
		err := errors.Wrap(err, "CreateSettings")
		t.Fatal(err)
	}

	taskSettings2 := models.TaskSettings{
		Name:             periodJobName2,
		PeriodInSeconds:  1,
		TimeoutInSeconds: 2,
		Status:           int(runner.TaskStatusIdle),
		LastFinished:     time.Now().Add(-time.Minute), // хотим запуск на первом тике
	}
	if err := settingsManager.CreateSettings(taskSettings2); err != nil {
		err := errors.Wrap(err, "CreateSettings")
		t.Fatal(err)
	}

	taskManager.Start()
	time.Sleep(time.Second*4 + time.Millisecond*700)
	taskManager.Stop()

	if task.RunCount != 4 {
		t.Errorf("wrong run count: %d", task.RunCount)
	}
	if task2.RunCount != 3 {
		t.Errorf("wrong run count: %d", task2.RunCount)
	}
}

func TestJobManager_Start_Timeout(t *testing.T) {
	settingsManager := NewSettingsManager()
	task2 := NewTestPeriodTask(time.Hour)

	results := make(chan runner.TaskResult)
	go func() {
		for {
			r := <-results
			log.Info(r.String())
			if r.Err != context.DeadlineExceeded {
				t.Error(r.Err)
			}
		}
	}()

	taskManager := runner.GetTaskManager(settingsManager, results)
	taskManager.SetWaitTime(time.Second*2).
		AddTask(periodJobName2, time.Second, time.Second/2, task2)
	taskManager.Start()

	taskSettings := models.TaskSettings{
		Name:             periodJobName2,
		PeriodInSeconds:  1,
		TimeoutInSeconds: 1,
		Status:           int(runner.TaskStatusIdle),
		LastFinished:     time.Now().Add(-time.Minute), // хотим запуск на первом тике
	}
	if err := settingsManager.UpdateSettings(taskSettings); err != nil {
		err := errors.Wrap(err, "UpdateSettings")
		t.Fatal(err)
	}
	taskManager.Reload()

	time.Sleep(time.Second * 4)
	taskManager.Stop()

	if task2.RunCount != 0 {
		t.Errorf("wrong run count: %d", task2.RunCount)
	}

}
func TestJobManager_ExecTask(t *testing.T) {
	settingsManager := NewSettingsManager()
	task := NewTestPeriodTask(0)

	results := make(chan runner.TaskResult)
	go func() {
		for {
			r := <-results
			log.Info(r.String())
			if r.Err != nil {
				t.Error(r.Err.Error())
			}
		}
	}()

	taskManager := runner.GetTaskManager(settingsManager, results)
	taskManager.AddTask(periodJobName, time.Minute*2, time.Minute*2, task)
	taskManager.SetWaitTime(time.Second)

	taskManager.Start()
	defer taskManager.Stop()

	newTaskManger := runner.GetTaskManager(settingsManager, results)
	if err := newTaskManger.ExecTask(periodJobName); err != nil {
		t.Error(err.Error())
	}
	time.Sleep(time.Millisecond * 100)

	if task.RunCount != 1 {
		t.Errorf("wrong run count: %d", task.RunCount)
	}
}
