package runner

import (
	"boogieman/src/model"
	"context"
	"github.com/enriquebris/goconcurrentqueue"
	"log"
	"sync"
	"time"
)

type Runner struct {
	script            *model.Script
	Progress          Progress
	probesStayedAlive *goconcurrentqueue.FIFO
}

type Progress struct {
	Idx             int
	RunnerStartedAt time.Time
	Check           *model.Task
	CheckStartedAt  time.Time
}

func NewRunner(script *model.Script) Runner {
	return Runner{
		script:            script,
		probesStayedAlive: goconcurrentqueue.NewFIFO(),
	}
}

// Run starts the script and blocks until finish
func (r *Runner) Run(ctx context.Context) {
	if err := r.script.EStatusRun(); err != nil {
		r.Log(err.Error())
		return
	}

	for _, cGroup := range r.script.CGroups {
		r.runCgroup(ctx, cGroup)
	}

	// finished
	succ := true
	for _, t := range r.script.Tasks {
		taskResult, _ := t.Result()
		succ = succ && taskResult.Success
	}

	_ = r.script.EStatusFinish(succ)

	// finishing background probes stayed alive
	for i, e := r.probesStayedAlive.Dequeue(); e == nil; i, e = r.probesStayedAlive.Dequeue() {
		probe, ok := i.(model.Prober)
		if !ok {
			r.Log("wrong queue object type")
			continue
		}
		probe.Finish(ctx)
	}
}

func (r *Runner) Result() model.ScriptResult {
	return r.script.Result()
}

func (r *Runner) runCgroup(ctx context.Context, cgroup *model.CGroup) (succ bool) {
	if err := cgroup.EStatusRun(); err != nil {
		r.Log("[cgroup][%v] %v", cgroup.Name, err.Error())
		return
	}

	ctx = context.WithValue(ctx, "cgroup", cgroup.Name)

	var wg sync.WaitGroup
	for _, task := range cgroup.Tasks {
		wg.Add(1)
		go func(task *model.Task) {
			defer func() {
				wg.Done()
			}()

			_, err := task.Start(ctx)
			if err != nil {
				r.Log("[task][%v] %v", task.Name, err.Error())
			}

			if task.Probe.IsAlive() {
				_ = r.probesStayedAlive.Enqueue(task.Probe)
			}
		}(task)
	}
	wg.Wait()
	succ = true
	for _, t := range r.script.Tasks {
		taskResult, _ := t.Result()
		succ = succ && taskResult.Success
	}
	_ = cgroup.EStatusFinish(succ)
	return
}

func (r *Runner) Log(format string, args ...any) {
	log.Printf("[runner]"+format+"\n", args...)
}
