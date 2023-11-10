package model

import (
	"github.com/starshiptroopers/uidgenerator"
	"time"
)

type Action string

const (
	ActionStart  = "start"
	ActionFinish = "finish"
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
	Daemon          bool
	Timeout         time.Duration
	Tasks           []*Task
	CGroups         []*CGroup
	anonymousCGroup *CGroup
}

type Task struct {
	Name   string
	CGroup string
	Probe  Prober
	Action Action
	//DependsOn      *Task
}

type CGroup struct {
	Name  string
	Tasks []*Task
}

func (c *CGroup) addTask(task *Task) {
	c.Tasks = append(c.Tasks, task)
}

func (s *Script) newCGroup(name string) (c *CGroup) {
	c = &CGroup{}
	if name != "" {
		c.Name = name
	} else {
		c.Name = uidGenerator.New()
	}

	s.CGroups = append(s.CGroups, c)
	return
}

func (s *Script) AddTask(t *Task) {
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

func NewTask(taskName string, cgroup string, probe Prober) (t *Task) {
	t = &Task{
		Name:   taskName,
		CGroup: cgroup,
		Probe:  probe,
	}
	return
}
