package model

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/creasty/defaults"
	"log"
	"strings"
	"sync"
	"time"
)

func init() {
	defaults.MustSet(&DefaultProbeOptions)
}

var ErrorConfig = errors.New("wrong configuration")

var DefaultProbeOptions ProbeOptions

type ProbeOptions struct {
	Timeout   time.Duration `json:"timeout,omitempty" default:"5000ms"`
	StayAlive bool          `json:"stayAlive,omitempty"` // probe process stays live after runner is exit, and should be finished by calling Finish method
	Expect    bool          `json:"expect,omitempty" default:"true"`
}

func (s *ProbeOptions) UnmarshalJSON(b []byte) (err error) {
	// init default options than parse json
	o := DefaultProbeOptions

	type tmp ProbeOptions
	t := tmp(o)

	err = json.Unmarshal(b, &t)
	if err != nil {
		return
	}
	t.Timeout = t.Timeout * time.Millisecond
	*s = ProbeOptions(t)

	return
}

type ProbeRunner func(ctx context.Context) (succ bool)

type ProbeHandler struct {
	ProbeOptions
	Name string // probe name
	//Configuration any
	runner     ProbeRunner              // probe runner
	startedAt  time.Time                // last startup timestamp
	duration   time.Duration            // last run duration
	finished   bool                     // probing is finished
	success    bool                     // probing is success
	timings    map[string]time.Duration // timings data
	logContext string                   // log prefix string
	error      error                    // last startup error
	sync.Mutex
}

// Start starts the probing and returns a probing result
func (c *ProbeHandler) Start(ctx context.Context) (succ bool) {
	if c.runner == nil {
		c.Log("runner isn't defined")
		return
	}
	c.startedAt = time.Now()
	c.finished = false
	c.success = false
	c.duration = 0
	if ctx != nil && ctx.Value("id") != nil {
		c.SetLogContext(ctx.Value("id").(string))
	}
	return c.runner(ctx)
}

// Duration returns duration of current or last finished probing
func (c *ProbeHandler) Duration() time.Duration {
	if c.duration == 0 && c.startedAt != (time.Time{}) {
		return time.Since(c.startedAt)
	}
	return c.duration
}

// Finished fixes the result of probing, should be called internally from the probe
func (c *ProbeHandler) Finished(success bool) {
	c.finished = true
	c.success = success
	if success {
		c.Log("SUCCESS")
	} else {
		c.Log("FAIL")
	}

	c.SetLogContext("")
	if c.startedAt != (time.Time{}) {
		c.duration = time.Since(c.startedAt)
	} else {
		c.duration = 0
	}
}

// SetDuration sets probing duration, should be called internally from the probe
func (c *ProbeHandler) SetDuration(d time.Duration) *ProbeHandler {
	c.duration = d
	return c
}

// SetError sets probing error, should be called internally from the probe
func (c *ProbeHandler) SetError(err error) *ProbeHandler {
	c.error = err
	return c
}

// SetRunner sets the probe runner, should be called internally from the probe on init
func (c *ProbeHandler) SetRunner(r ProbeRunner) *ProbeHandler {
	c.runner = r
	return c
}

// SetTimeStat saves probing statistics, should be called internally from the probe
func (c *ProbeHandler) SetTimeStat(name string, dur time.Duration) {
	c.Lock()
	if c.timings == nil {
		c.timings = make(map[string]time.Duration)
	}
	c.timings[name] = dur
	c.Unlock()
}

// TimeStat returns probing stat data
func (c *ProbeHandler) TimeStat() map[string]time.Duration {
	if c.timings == nil {
		return make(map[string]time.Duration)
	}
	return c.timings
}

// Log logouts the message, should be called internally from the probe
func (c *ProbeHandler) Log(format string, args ...any) {
	if c.logContext == "" {
		c.SetLogContext("")
	}
	var delim string
	if !strings.HasPrefix(format, "[") {
		delim = " "
	}
	log.Printf(c.logContext+delim+format+"\n", args...)
}

// SetLogContext sets log context, should be called internally from the probe
func (c *ProbeHandler) SetLogContext(ctx string) {
	if ctx != "" {
		c.logContext = "[" + ctx + "][" + c.Name + "]"
	} else {
		c.logContext = "[" + c.Name + "]"
	}
}

// Finish is just a stub for probe that should fit the Prober interface but doesn't have a Finish method itself
func (c *ProbeHandler) Finish() {

}

func (c *ProbeHandler) Error() error {
	return c.error
}

type Prober interface {
	Start(ctx context.Context) (succ bool)
	Duration() time.Duration
	TimeStat() map[string]time.Duration
	Error() error
	//Finished(success bool)
	Finish()
}
