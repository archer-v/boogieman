package model

import (
	"context"
	"github.com/enriquebris/goconcurrentqueue"
	"github.com/starshiptroopers/uidgenerator"
	"sync"
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
	anonymousCGroup   *CGroup
	probesStayedAlive *goconcurrentqueue.FIFO
}

type ScriptResult struct {
	Result `json:"result"`
	Status string       `json:"status"`
	Tasks  []TaskResult `json:"tasks"`
}

// Run starts the script and blocks until finish
func (s *Script) Run(ctx context.Context) {
	if s.probesStayedAlive == nil {
		s.probesStayedAlive = goconcurrentqueue.NewFIFO()
	}
	if err := s.EStatusRun(); err != nil {
		s.log(err.Error())
		return
	}

	for _, cGroup := range s.CGroups {
		s.runCgroup(ctx, cGroup)
		select {
		case <-ctx.Done():
		default:
		}
	}

	// finished
	succ := true
	for _, t := range s.Tasks {
		taskResult, _ := t.Worker.Result()
		succ = succ && taskResult.Success
	}

	_ = s.EStatusFinish(succ)

	// finishing background probes stayed alive
	for i, e := s.probesStayedAlive.Dequeue(); e == nil; i, e = s.probesStayedAlive.Dequeue() {
		probe, ok := i.(Prober)
		if !ok {
			s.log("wrong queue object type")
			continue
		}
		probe.Finish(ctx)
	}
}

func (s *Script) AddTask(t *Task) {
	if s.EStatus != EStatusNew {
		return
	}
	// task without concurrent group will be assigned to a new created default concurrent group
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
		r.Tasks[i] = t.Result()
	}
	return
}

func (s *Script) ResultFinished() (r ScriptResult) {
	r.Result = s.Worker.ResultFinished()
	r.Status = string(EStatusFinished)
	r.Tasks = make([]TaskResult, len(s.Tasks))
	for i, t := range s.Tasks {
		r.Tasks[i] = t.ResultFinished()
	}
	return
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

func (s *Script) runCgroup(ctx context.Context, cgroup *CGroup) (succ bool) {
	if err := cgroup.EStatusRun(); err != nil {
		s.log("[cgroup][%v] %v", cgroup.Name, err.Error())
		return
	}

	ctx = context.WithValue(ctx, "cgroup", cgroup.Name)

	var wg sync.WaitGroup
	for _, task := range cgroup.Tasks {
		wg.Add(1)
		go func(task *Task) {
			defer func() {
				wg.Done()
			}()

			_, err := task.Start(ctx)
			if err != nil {
				s.log("[task][%v] %v", task.Name, err.Error())
			}

			if task.Probe.IsAlive() {
				_ = s.probesStayedAlive.Enqueue(task.Probe)
			}
		}(task)
	}
	wg.Wait()
	succ = true
	for _, t := range s.Tasks {
		taskResult, _ := t.Worker.Result()
		succ = succ && taskResult.Success
	}
	_ = cgroup.EStatusFinish(succ)
	return
}

func (s *Script) log(format string, args ...any) {
	Logger.Printf("[runner]"+format+"\n", args...)
}
