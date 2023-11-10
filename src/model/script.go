package model

import (
	"context"
	"errors"
	"fmt"
	"github.com/starshiptroopers/uidgenerator"
	"sync"
	"time"
)

type EStatus string

const (
	EStatusFinished EStatus = "finished"
	EStatusNew      EStatus = "new"
	EStatusRunning  EStatus = "running"
)

var uidGenerator *uidgenerator.UIDGenerator

func init() {
	conf := uidgenerator.Cfg{
		Alfa:      "1234567890abcdef",
		Format:    "XXXXX",
		Validator: "[0-9a-zA-Z]{5}",
	}
	uidGenerator = uidgenerator.New(
		&conf,
	)
}

type Script struct {
	Daemon  bool
	Timeout time.Duration
	Tasks   []*Task
	CGroups []*CGroup
	Runner
	anonymousCGroup *CGroup
}

type Task struct {
	Name   string
	CGroup string
	Probe  Prober
	Runner
	//DependsOn      *Task
}

func (t *Task) Start(ctx context.Context) (succ bool, err error) {
	if err = t.EStatusRun(); err != nil {
		return
	}
	succ = t.Probe.Start(context.WithValue(ctx, "id", t.Name))
	err = t.EStatusFinish(succ)
	return
}

type Result struct {
	StartedAt time.Time
	Duration  time.Duration
	Success   bool
}

type CGroup struct {
	Name string
	Runner
	Tasks []*Task
}

type Runner struct {
	Result
	EStatus
	sync.Mutex
}

func (c *CGroup) addTask(task *Task) {
	c.Tasks = append(c.Tasks, task)
}

func (s *Script) newCGroup(name string) (c *CGroup) {
	c = &CGroup{
		Runner: Runner{
			EStatus: EStatusNew,
		},
	}
	if name != "" {
		c.Name = name
	} else {
		c.Name = uidGenerator.New()
	}

	s.CGroups = append(s.CGroups, c)
	return
}

func (s *Script) AddTask(t *Task) {
	if s.EStatus != EStatusNew {
		return
	}
	// task without defined concurrent group will be assigned to a new created default concurrent group
	if t.CGroup == "" {
		if s.anonymousCGroup == nil || (len(s.Tasks) > 0 && s.Tasks[len(s.Tasks)-1].CGroup != s.anonymousCGroup.Name) {
			s.anonymousCGroup = s.newCGroup("")
		}
		t.CGroup = s.anonymousCGroup.Name
		s.anonymousCGroup.addTask(t)
	} else {
		if len(s.CGroups) == 0 || s.CGroups[len(s.CGroups)-1].Name != t.CGroup {
			cgroup := s.newCGroup(t.CGroup)
			cgroup.addTask(t)
		} else {
			s.CGroups[len(s.CGroups)-1].addTask(t)
		}
	}
	s.Tasks = append(s.Tasks, t)
}

func (s *Runner) EStatusRun() (err error) {
	s.Lock()
	defer s.Unlock()
	if s.EStatus == EStatusRunning {
		return errors.New("already " + string(s.EStatus))
	}
	s.EStatus = EStatusRunning
	s.Success = false
	s.StartedAt = time.Now()
	s.Duration = 0
	return
}

func (s *Runner) EStatusFinish(succ bool) (err error) {
	s.Lock()
	defer s.Unlock()
	if s.EStatus != EStatusRunning {
		return fmt.Errorf("can't switch from status %v to %v", string(s.EStatus), string(EStatusFinished))
	}
	s.EStatus = EStatusFinished
	s.Duration = time.Since(s.StartedAt)
	s.Success = succ
	return
}

func NewTask(taskName string, cgroup string, probe Prober) (t *Task) {
	t = &Task{
		Name:   taskName,
		CGroup: cgroup,
		Probe:  probe,
		Runner: Runner{
			EStatus: EStatusNew,
		},
	}
	return
}

func NewScript() (s Script) {
	s = Script{
		Runner: Runner{
			EStatus: EStatusNew,
		},
	}
	return
}
