package model

import "time"

type Action string

const (
	ActionStart  = "start"
	ActionFinish = "finish"
)

type Script struct {
	Daemon  bool
	Timeout time.Duration
	Tasks   []*Task
}

type Task struct {
	Name         string
	Probe        Prober
	Action       Action
	Dependencies []string
}
