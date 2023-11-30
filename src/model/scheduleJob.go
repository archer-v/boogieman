package model

import (
	"github.com/go-co-op/gocron"
	"time"
)

type ScheduleJob struct {
	Name       string        `json:"name"`
	ScriptFile string        `json:"script"`
	Schedule   string        `json:"schedule"`
	Once       bool          `json:"once"`
	Timeout    time.Duration `json:"timeout"`
	Script     *Script       `json:"-"`
	CronJob    *gocron.Job   `json:"-"`
}
