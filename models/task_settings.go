package models

import (
	"time"
)

type TaskSettings struct {
	Name             string
	PeriodInSeconds  int64
	TimeoutInSeconds int64
	Status           int
	UpdatedAt        time.Time
	LastFinished     time.Time
	LastStarted      time.Time
}

type TasksSettingsData struct {
	Tasks JobSettingsViews `json:"tasks_period"`
}

type TaskSettingsView struct {
	Name         string `json:"name"`
	Period       int64  `json:"period"`
	Status       string `json:"status"`
	LastFinished string `json:"last_finished"`
	LastStarted  string `json:"last_started"`
	Timeout      int64  `json:"timeout"`
}

type JobSettingsViews []TaskSettingsView

func (j JobSettingsViews) Len() int {
	return len(j)
}

func (j JobSettingsViews) Less(i, k int) bool {
	return j[i].Name < j[k].Name
}

func (j JobSettingsViews) Swap(i, k int) {
	j[i], j[k] = j[k], j[i]
}
