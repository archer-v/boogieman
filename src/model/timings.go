package model

import (
	"sync"
	"time"
)

type Timings struct {
	Timings map[string]time.Duration
	sync.Mutex
}

func (c *Timings) Set(name string, dur time.Duration) {
	c.Lock()
	if c.Timings == nil {
		c.Timings = make(map[string]time.Duration)
	}
	c.Timings[name] = dur
	c.Unlock()
}
