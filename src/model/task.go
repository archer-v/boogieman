package model

import "context"

type Task struct {
	Name   string
	CGroup string `json:"-"`
	Probe  Prober
	Worker
	runCounter uint
}

type TaskResult struct {
	Name       string
	ID         uint
	RunCounter uint
	Result
	Status string
	Probe  ProbeResult
}

func (t *Task) Start(ctx context.Context) (succ bool, err error) {
	t.runCounter++
	if err = t.EStatusRun(t.runCounter); err != nil {
		return
	}
	succ = t.Probe.Start(context.WithValue(ctx, "id", t.Name))
	err = t.EStatusFinish(succ)
	return
}

func (t *Task) RunCounter() uint {
	return t.runCounter
}

func NewTask(taskName string, cgroup string, probe Prober) (t *Task) {
	t = &Task{
		Name:   taskName,
		CGroup: cgroup,
		Probe:  probe,
		Worker: Worker{
			EStatus: EStatusNew,
		},
	}
	return
}
