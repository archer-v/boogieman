package runner

import (
	"boogieman/src/model"
	"context"
	"log"
	"sync"
	"time"
)

type Runner struct {
	script   model.Script
	Result   ScriptResult
	Progress Progress
}

type ScriptResult struct {
	StartedAt time.Time
	Duration  time.Duration
	Checks    []CheckResult
	Success   bool
}

type CheckResult struct {
	Name     string
	Success  bool
	Duration time.Duration
	Timings  map[string]time.Duration
}

type Progress struct {
	Idx             int
	RunnerStartedAt time.Time
	Check           *model.Task
	CheckStartedAt  time.Time
}

func NewRunner(script model.Script) Runner {
	return Runner{
		script: script,
	}
}

func (r *Runner) Run(ctx context.Context) {
	if err := r.script.EStatusRun(); err != nil {
		r.Log(err.Error())
		return
	}

	for _, cgroup := range r.script.CGroups {
		r.runCgroup(ctx, cgroup)
	}

	succ := true
	for _, t := range r.script.Tasks {
		succ = succ && t.Success
	}

	_ = r.script.EStatusFinish(succ)
}

func (r *Runner) runCgroup(ctx context.Context, cgroup *model.CGroup) (succ bool) {
	if err := cgroup.EStatusRun(); err != nil {
		r.Log("[cgroup][%v] %v", cgroup.Name, err.Error())
		return
	}

	ctx = context.WithValue(ctx, "cgroup", cgroup.Name)
	//r.Progress.RunnerStartedAt = time.Now()
	//r.Result.Checks = make([]CheckResult, len(r.script.Tasks))
	var wg sync.WaitGroup
	for _, task := range cgroup.Tasks {

		/*
			//r.Progress.Idx = i
			//r.Progress.Check = task
			//r.Progress.CheckStartedAt = time.Now()

			r.Result.Checks[i] = CheckResult{
				Name:     task.Name,
				Runtime: 0,
				Timings:  nil,
			}
		*/
		wg.Add(1)
		go func(task *model.Task) {
			defer func() {
				wg.Done()
			}()

			_, err := task.Start(ctx)
			if err != nil {
				r.Log("[task][%v] %v", task.Name, err.Error())
			}
			//task.Probe.Finish()
		}(task)

		//r.Result.Checks[i].Timings = task.Probe.TimeStat()
	}
	wg.Wait()
	succ = true
	for _, t := range r.script.Tasks {
		succ = succ && t.Success
	}
	return
}

func (r *Runner) Log(format string, args ...any) {
	log.Printf("[runner]"+format+"\n", args...)
}
