package tests

import (
	"database/sql"
	"github.com/0ndi/go-task-manager/models"
	"github.com/pkg/errors"
	"time"
)

var TaskData map[string]models.TaskSettings = map[string]models.TaskSettings{}

type mockSettingsManager struct {}

func NewSettingsManager() *mockSettingsManager {
	TaskData = map[string]models.TaskSettings{}
	return &mockSettingsManager{}
}

func (m mockSettingsManager) GetSettings(s string) (models.TaskSettings, error) {
	task, ok := TaskData[s]
	if !ok {
		return task, sql.ErrNoRows
	}
	return task, nil
}

func (m mockSettingsManager) CreateSettings(settings models.TaskSettings) error {
	if _, exist := TaskData[settings.Name]; exist {
		return errors.New("already exist")
	}
	TaskData[settings.Name] = settings
	return nil
}

func (m mockSettingsManager) UpdateSettings(settings models.TaskSettings) error {
	if _, exist := TaskData[settings.Name]; !exist {
		return sql.ErrNoRows
	}
	TaskData[settings.Name] = settings
	return nil
}

func (m mockSettingsManager) UpdateSettingsTime(taskName string, start time.Time, finish time.Time) error {
	if _, exist := TaskData[taskName]; !exist {
		return sql.ErrNoRows
	}
	return nil
}

