package model

import "context"

type Task struct {
	Name   string
	CGroup string     `json:"-"`
	Metric TaskMetric `json:"-"`
	Probe  Prober
	Worker
}

type TaskResult struct {
	Name   string      `json:"name"`
	Status string      `json:"status"`
	Probe  ProbeResult `json:"probe"`
	Result
}

func (t *Task) Start(ctx context.Context) (succ bool, err error) {
	if err = t.EStatusRun(); err != nil {
		return
	}
	succ = t.Probe.Start(ContextWithLogger(ctx, NewChainLogger(GetLogger(ctx), t.Name)))
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
	if tr.Result.Completed() {
		tr.Status = string(EStatusFinished)
	} else {
		tr.Status = string(EStatusNew)
	}
	tr.Probe = t.Probe.ResultFinished()
	return
}

func NewTask(taskName string, cgroup string, metric TaskMetric, probe Prober) (t *Task) {
	t = &Task{
		Name:   taskName,
		CGroup: cgroup,
		Metric: metric,
		Probe:  probe,
		Worker: Worker{
			EStatus: EStatusNew,
		},
	}
	return
}
