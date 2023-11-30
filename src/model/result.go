package model

import (
	"time"
)

type Result struct {
	StartedAt  time.Time     `json:"startedAt"`
	Runtime    time.Duration `json:"-"`
	RuntimeMs  int           `json:"runtime"`
	Success    bool          `json:"success"`
	RunCounter uint          `json:"runCounter"` // run counter
}

func (r *Result) PrepareToStart() {
	r.Success = false
	r.StartedAt = time.Now()
	r.Runtime = 0
	r.RuntimeMs = 0
	r.RunCounter++
}

func (r *Result) End(succ bool) {
	r.Runtime = time.Since(r.StartedAt)
	r.RuntimeMs = int(r.Runtime.Milliseconds())
	r.Success = succ
}

func (r *Result) Completed() bool {
	return r.Runtime != 0
}
