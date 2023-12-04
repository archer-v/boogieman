package model

import (
	"github.com/go-co-op/gocron"
	"time"
)

type ScheduleJob struct {
	Name        string        `json:"name"`
	ScriptFile  string        `json:"script"`
	Schedule    string        `json:"schedule"`
	Once        bool          `json:"once"`
	Timeout     time.Duration `json:"timeout"`
	NextStartAt time.Time     `json:"nextStartAt"` // exclusively for JSON export
	Script      *Script       `json:"-"`
	CronJob     *gocron.Job   `json:"-"`
	Vars        map[string]map[string]string
}
