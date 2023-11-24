package model

import "context"

type Task struct {
	Name   string
	CGroup string `json:"-"`
	Probe  Prober
	Worker
	//DependsOn      *Task
}

type TaskResult struct {
	Name string
	Result
	Status string
	Probe  ProbeResult
}

func (t *Task) Start(ctx context.Context) (succ bool, err error) {
	if err = t.EStatusRun(); err != nil {
		return
	}
	succ = t.Probe.Start(context.WithValue(ctx, "id", t.Name))
	err = t.EStatusFinish(succ)
	return
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
