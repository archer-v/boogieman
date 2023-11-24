package model

import "time"

type Result struct {
	StartedAt time.Time
	Runtime   time.Duration
	Success   bool
}

func (r *Result) PrepareToStart() {
	r.Success = false
	r.StartedAt = time.Now()
	r.Runtime = 0
}

func (r *Result) End(succ bool) {
	r.Runtime = time.Since(r.StartedAt)
	r.Success = succ
}

func (r *Result) Completed() bool {
	return r.Runtime != 0
}
