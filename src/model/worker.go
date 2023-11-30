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

type Worker struct {
	prevResult Result
	curResult  Result
	EStatus
	sync.Mutex
}

func (s *Worker) EStatusRun() (err error) {
	s.Lock()
	defer s.Unlock()
	if s.EStatus == EStatusRunning {
		return errors.New("already " + string(s.EStatus))
	}
	s.EStatus = EStatusRunning
	s.curResult.PrepareToStart()
	return
}

func (s *Worker) EStatusFinish(succ bool) (err error) {
	s.Lock()
	defer s.Unlock()
	if s.EStatus != EStatusRunning {
		return fmt.Errorf("can't switch from status %v to %v", string(s.EStatus), string(EStatusFinished))
	}
	s.EStatus = EStatusFinished
	s.curResult.End(succ)
	s.prevResult = s.curResult
	return
}

func (s *Worker) Duration() time.Duration {
	if s.curResult.Runtime == 0 && s.curResult.StartedAt != (time.Time{}) {
		return time.Since(s.curResult.StartedAt)
	}
	return s.curResult.Runtime
}

// Result returns a current worker status and a lastResult
func (s *Worker) Result() (result Result, status EStatus) {
	s.Lock()
	defer s.Unlock()
	status = s.EStatus
	result = s.curResult
	return
}

// ResultFinished returns a last finished worker status and a lastResult
func (s *Worker) ResultFinished() (result Result) {
	s.Lock()
	defer s.Unlock()
	result = s.prevResult
	return
}
