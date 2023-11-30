package model

import "context"

type Task struct {
	Name   string
	CGroup string `json:"-"`
	Probe  Prober
	Worker
}

type TaskResult struct {
	Name   string `json:"name"`
	Result `json:"result"`
	Status string      `json:"status"`
	Probe  ProbeResult `json:"probe"`
}

func (t *Task) Start(ctx context.Context) (succ bool, err error) {
	if err = t.EStatusRun(); err != nil {
		return
	}
	succ = t.Probe.Start(context.WithValue(ctx, "id", t.Name))
	err = t.EStatusFinish(succ)
	return
}

func (t *Task) Result() (tr TaskResult) {
	tr.Name = t.Name
	trr, trs := t.Worker.Result()
	tr.Result = trr
	tr.Status = string(trs)
	tr.Probe = t.Probe.Result()
	return
}

func (t *Task) ResultFinished() (tr TaskResult) {
	tr.Name = t.Name
	tr.Result = t.Worker.ResultFinished()
	tr.Status = string(EStatusFinished)
	tr.Probe = t.Probe.ResultFinished()
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
