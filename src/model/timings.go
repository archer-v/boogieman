package model

import (
	"encoding/json"
	"sync"
	"time"
)

type Timings struct {
	Timings map[string]time.Duration `json:"timings"`
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

func (c *Timings) MarshalJSON() ([]byte, error) {
	type timingsMillis map[string]int
	tm := make(timingsMillis)
	for k, v := range c.Timings {
		tm[k] = int(v.Milliseconds())
	}
	return json.Marshal(tm)
}
