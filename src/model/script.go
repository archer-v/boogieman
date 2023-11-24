package model

import (
	"github.com/starshiptroopers/uidgenerator"
	"time"
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
	Timeout time.Duration `json:"-"`
	Tasks   []*Task
	CGroups []*CGroup `json:"-"`
	Worker
	anonymousCGroup *CGroup
}

type ScriptResult struct {
	Result
	Status string
	Tasks  []TaskResult
}

func (s *Script) newCGroup(name string) (c *CGroup) {
	c = &CGroup{
		Worker: Worker{
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

func (s *Script) Result() (r ScriptResult) {
	rr, rs := s.Worker.Result()
	r.Result = rr
	r.Status = string(rs)
	r.Tasks = make([]TaskResult, len(s.Tasks))
	for i, t := range s.Tasks {
		tr := TaskResult{}
		tr.Name = t.Name
		trr, trs := t.Result()
		tr.Result = trr
		tr.Status = string(trs)
		tr.Probe = t.Probe.Result()
		r.Tasks[i] = tr
	}
	return
}

func (s *Script) ResultFinished() (r ScriptResult) {
	r.Result = s.Worker.ResultFinished()
	r.Status = string(EStatusFinished)
	r.Tasks = make([]TaskResult, len(s.Tasks))
	for i, t := range s.Tasks {
		tr := TaskResult{}
		tr.Name = t.Name
		tr.Result = t.ResultFinished()
		tr.Status = string(EStatusFinished)
		tr.Probe = t.Probe.ResultFinished()
		r.Tasks[i] = tr
	}
	return
}
