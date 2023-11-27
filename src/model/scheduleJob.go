package model

import (
	"github.com/go-co-op/gocron"
	"time"
)

type ScheduleJob struct {
	Name       string
	ScriptFile string `json:"script"`
	Schedule   string
	Once       bool
	Timeout    time.Duration
	Script     *Script
	CronJob    *gocron.Job
}
