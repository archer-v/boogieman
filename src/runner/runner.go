package runner

import (
	"context"
	"liberator-check/src/model"
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
	r.Result.Success = false
	r.Progress.RunnerStartedAt = time.Now()
	r.Result.Checks = make([]CheckResult, len(r.script.Tasks))
	for i, task := range r.script.Tasks {
		r.Progress.Idx = i
		r.Progress.Check = task
		r.Progress.CheckStartedAt = time.Now()

		r.Result.Checks[i] = CheckResult{
			Name:     task.Name,
			Duration: 0,
			Timings:  nil,
		}
		r.Result.Checks[i].Success = task.Probe.Start(context.WithValue(ctx, "id", task.Name))
		r.Result.Checks[i].Duration = time.Since(r.Progress.CheckStartedAt)
		r.Result.Checks[i].Timings = task.Probe.TimeStat()
	}

	r.Result.Success = true
	for _, rz := range r.Result.Checks {
		r.Result.Success = r.Result.Success && rz.Success
	}
}
