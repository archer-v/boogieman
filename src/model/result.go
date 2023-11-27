package model

import "time"

type Result struct {
	ID        uint
	StartedAt time.Time
	Runtime   time.Duration
	Success   bool
}

func (r *Result) PrepareToStart(id ...uint) {
	if len(id) > 0 {
		r.ID = id[0]
	} else {
		r.ID = 0
	}

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
