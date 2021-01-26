package runner

import (
	"context"
	"database/sql"
	"github.com/0ndi/go-task-manager/models"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"sort"
	"sync"
	"time"
)

const (
	viewTimeFormat = "15:04:05 02.01"
)

var (
	manager         *TaskManager
	mut             sync.Mutex
	defaultWaitTime = time.Second * 9

	TaskNotFoundError       = errors.New("Task not found")
	TaskAlreadyRunningError = errors.New("Task already running, wait until complete and try again")
)

type TaskSettingsStorage interface {
	GetSettings(string) (models.TaskSettings, error)
	CreateSettings(models.TaskSettings) error
	UpdateSettings(models.TaskSettings) error
	UpdateSettingsTime(string, time.Time, time.Time) error
}

type TaskManager struct {
	reloadChan          chan struct{}
	tasks               map[string]*managedTask
	taskResults         chan TaskResult
	taskSettingsManager TaskSettingsStorage
	shutdownChan        chan struct{}
	waitAfterStop       time.Duration
}

func GetTaskManager(settingsManager TaskSettingsStorage, results chan TaskResult) *TaskManager {
	mut.Lock()
	defer mut.Unlock()

	if manager == nil {
		manager = newTaskManager(settingsManager, results)
	}
	return manager
}

func newTaskManager(settingsManager TaskSettingsStorage, results chan TaskResult) *TaskManager {
	manager := TaskManager{}

	manager.tasks = make(map[string]*managedTask)
	manager.reloadChan = make(chan struct{})
	manager.shutdownChan = make(chan struct{})
	manager.waitAfterStop = defaultWaitTime
	manager.taskResults = results

	manager.taskSettingsManager = settingsManager

	return &manager
}

func (m *TaskManager) GetView() *models.TasksSettingsData {
	var tasks models.JobSettingsViews
	for _, taskPtr := range m.tasks {
		task := *taskPtr

		var status string
		switch task.status {
		case TaskStatusIdle:
			status = TaskViewStatusIdle
		case TaskStatusDisabled:
			status = TaskViewStatusDisabled
		case TaskStatusRunning:
			status = TaskViewStatusRunning
		}

		taskView := models.TaskSettingsView{
			Name:         task.name,
			Period:       int64(task.period / time.Second),
			Timeout:      int64(task.timeout / time.Second),
			Status:       status,
			LastFinished: task.lastFinished.Format(viewTimeFormat),
			LastStarted:  task.lastStarted.Format(viewTimeFormat),
		}

		tasks = append(tasks, taskView)
	}

	sort.Sort(tasks)

	data := models.TasksSettingsData{
		Tasks: tasks,
	}
	return &data
}

func (m *TaskManager) SetWaitTime(wait time.Duration) *TaskManager {
	m.waitAfterStop = wait
	return m
}

func (m *TaskManager) Reload() {
	m.reloadChan <- struct{}{}
}

func (m *TaskManager) Stop() {
	m.shutdownChan <- struct{}{}
	time.Sleep(m.waitAfterStop)
	//
	//close(m.reloadChan)
	//close(m.shutdownChan)
	//log.Info("Channels are closed")
}

func (m *TaskManager) AddTask(name string, period, timeout time.Duration, task Task) *TaskManager {
	job := newPeriodicTask(name, period, timeout, task)
	m.tasks[name] = job
	return m
}

func (m *TaskManager) Start() {
	m.updateConfig()
	go m.runTasks()
}

func (m *TaskManager) ExecTask(name string) error {
	task, ok := m.tasks[name]
	if !ok {
		return TaskNotFoundError
	}

	if task.status == TaskStatusRunning {
		return TaskAlreadyRunningError
	}

	go m.runTask(task)
	return nil
}

func (m *TaskManager) runTasks() {
	tickerSecond := time.NewTicker(time.Second)
	defer tickerSecond.Stop()
	defer log.Info("runTasks loop stop")

	for {
		select {
		case <-tickerSecond.C:
			m.proceed()
		case <-m.reloadChan:
			//Если пришло сообщение обновляем конфиг
			m.updateConfig()
		case <-m.shutdownChan:
			log.Info("Shutdown signal. Stop")
			//Если пришло сообщение выходим из функции, тем самым останавливаем раннер
			return
		}

	}
}

func (m *TaskManager) runTask(task *managedTask) {
	ctx, cancel := context.WithTimeout(context.Background(), task.timeout)
	defer cancel()

	//Запоминаем изначальное состояние таска, если он был выключен и его запустили через ExecTask, по завршению таска
	// он должен вернуться в выключенное состояние. Аналогично для idle таска
	initialState := task.status
	task.SetStatus(TaskStatusRunning)
	defer func() {
		// Если в процессе исполнения таск выключили, выставлять статус не нужно.
		// Меняем только если таск находится в статусе TaskStatusRunning
		if task.status == TaskStatusRunning {
			task.SetStatus(initialState)
		}
	}()

	taskResult := NewTaskResult()

	resultCh := make(chan *managedTaskResult, 1)
	go task.Execute(ctx, taskResult.UID, resultCh)

	var err error
	select {
	case <-ctx.Done():
		log.Errorf("task %s(%s) timeout(%d s)", task.name, taskResult.UID, task.timeout/time.Second)
		err = ctx.Err()
	case mtResult := <-resultCh:
		taskResult.RawResult = mtResult.RawResult
		err = mtResult.Err
	}

	finished := time.Now()

	taskResult.Finished = finished
	taskResult.Err = err
	taskResult.Name = task.name

	//отправляем результат на обработку
	m.taskResults <- *taskResult

	// Выставляем время последнего запуска, по нему ориентируется метод CanExecute
	task.setLastFinished(finished)

	if err := m.taskSettingsManager.UpdateSettingsTime(task.name, task.lastStarted, task.lastFinished); err != nil {
		err := errors.Wrap(err, "UpdateSettingsTime")
		log.Error(err)
	}
}

func (m *TaskManager) updateConfig() {
	log.Info("Update tasks settings started..")
	// идем по всем таскам, по каждой делаем запрос в бд
	for taskName, task := range m.tasks {
		taskSettings, err := m.taskSettingsManager.GetSettings(taskName)
		if err != nil {
			// если таскам с таким именем нет, нужно создать
			if err == sql.ErrNoRows {
				//save settings
				newStatus := TaskStatusDisabled
				switch task.status {
				case TaskStatusIdle, TaskStatusRunning:
					newStatus = TaskStatusIdle
				}

				newTaskSettings := models.TaskSettings{
					Name:             task.name,
					PeriodInSeconds:  int64(task.period / time.Second),
					TimeoutInSeconds: int64(task.timeout / time.Second),
					Status:           int(newStatus),
					UpdatedAt:        time.Now(),
					LastFinished:     task.lastFinished,
					LastStarted:      task.lastStarted,
				}
				if err := m.taskSettingsManager.CreateSettings(newTaskSettings); err != nil {
					err := errors.Wrap(err, "UpdateSettings")
					log.Errorf("%+v update task settings error: %s", newTaskSettings, err.Error())
				} else {
					log.Infof("New settings saved: %+v", newTaskSettings)
				}
			}

			err := errors.Wrap(err, "GetSettings")
			log.Error(err)
			continue
		}

		// валидация статуса
		newStatus := TaskStatus(taskSettings.Status)
		switch newStatus {
		case TaskStatusDisabled, TaskStatusIdle:
			task.status = newStatus
		}

		//todo hide it inside SetPeriod
		if taskSettings.PeriodInSeconds <= 0 {
			taskSettings.PeriodInSeconds = 1
		}
		task.period = time.Second * time.Duration(taskSettings.PeriodInSeconds)

		//todo hide it inside SetTimeout
		if taskSettings.TimeoutInSeconds <= 0 {
			taskSettings.TimeoutInSeconds = 1
		}
		task.timeout = time.Second * time.Duration(taskSettings.TimeoutInSeconds)
		if task.lastFinished.Before(taskSettings.LastFinished) {
			task.setLastFinished(taskSettings.LastFinished)
		}
	}
	log.Info("Tasks settings updated")
}

func (m *TaskManager) getIdleTask() ([]*managedTask, error) {
	var tasks []*managedTask
	for _, task := range m.tasks {
		if task.status == TaskStatusIdle {
			tasks = append(tasks, task)
		}
	}
	return tasks, nil
}

func (m *TaskManager) proceed() {
	tasks, err := m.getIdleTask()
	if err != nil {
		err := errors.Wrapf(err, "GetIdleTask")
		log.Error(err)
		return
	}

	for _, task := range tasks {
		if !task.CanExecute() {
			continue
		}
		go m.runTask(task)
	}
}
