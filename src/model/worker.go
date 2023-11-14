package model

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

type EStatus string

const (
	EStatusFinished EStatus = "finished"
	EStatusNew      EStatus = ""
	EStatusRunning  EStatus = "running"
)

type Runner struct {
	Result
	EStatus
	sync.Mutex
}

func (s *Runner) EStatusRun() (err error) {
	s.Lock()
	defer s.Unlock()
	if s.EStatus == EStatusRunning {
		return errors.New("already " + string(s.EStatus))
	}
	s.EStatus = EStatusRunning
	s.Result.PrepareToStart()
	return
}

func (s *Runner) EStatusFinish(succ bool) (err error) {
	s.Lock()
	defer s.Unlock()
	if s.EStatus != EStatusRunning {
		return fmt.Errorf("can't switch from status %v to %v", string(s.EStatus), string(EStatusFinished))
	}
	s.EStatus = EStatusFinished
	s.Result.End(succ)
	return
}

func (s *Runner) Duration() time.Duration {
	if s.Result.Runtime == 0 && s.StartedAt != (time.Time{}) {
		return time.Since(s.StartedAt)
	}
	return s.Result.Runtime
}

/*
type Worker struct {
	PollInterval time.Runtime
	CheckTimeout time.Runtime
	Machine      Machine
	//Checks       CheckPlan
}
*/
